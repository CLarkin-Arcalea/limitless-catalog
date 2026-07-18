package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

//go:embed static
var staticFiles embed.FS

// serveHandlers binds the read-only store to HTTP handlers. Same seven
// operations mcpHandlers exposes over MCP stdio, here over net/http.
type serveHandlers struct {
	s   *store.Store
	loc *time.Location
}

func (h serveHandlers) stats() (store.Stats, error) {
	return h.s.Stats("")
}

func (h serveHandlers) search(a searchArgs) (rowsOut, error) {
	limit := a.Limit
	if limit <= 0 {
		limit = 20
	}
	rows, err := h.s.Search(a.Query, limit)
	return rowsOut{Results: rows}, err
}

func (h serveHandlers) meeting(a meetingArgs) (rowsOut, error) {
	startT, err := parseLocalDateTime(a.Start, h.loc)
	if err != nil {
		return rowsOut{}, err
	}
	endT := startT.Add(time.Hour)
	if a.End != "" {
		endT, err = parseLocalDateTime(a.End, h.loc)
		if err != nil {
			return rowsOut{}, err
		}
	}
	buffer := a.BufferMin
	if buffer <= 0 {
		buffer = 10
	}
	rows, err := h.s.Meeting(startT, endT, time.Duration(buffer)*time.Minute)
	return rowsOut{Results: rows}, err
}

func (h serveHandlers) byDate(a dateArgs) (rowsOut, error) {
	rows, err := h.s.ByDate(a.Date)
	return rowsOut{Results: rows}, err
}

func (h serveHandlers) byRange(a rangeArgs) (rowsOut, error) {
	rows, err := h.s.ByRange(a.StartDate, a.EndDate)
	return rowsOut{Results: rows}, err
}

func (h serveHandlers) recent(a recentArgs) (rowsOut, error) {
	n := a.Count
	if n <= 0 {
		n = 10
	}
	rows, err := h.s.Recent(n)
	return rowsOut{Results: rows}, err
}

func (h serveHandlers) getTranscript(a getArgs) (recordOut, error) {
	fr, err := h.s.Get(a.ID)
	if err != nil {
		return recordOut{}, err
	}
	if fr == nil {
		return recordOut{}, fmt.Errorf("no lifelog with id %q", a.ID)
	}
	fr.RawJSON = ""
	if !a.Full {
		fr.TranscriptMD = ""
	}
	return recordOut{FullRecord: *fr}, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// atoiOr parses s as an int, returning def when s is empty or unparseable.
func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// newServeMux wires the seven read-only routes plus the embedded frontend.
func newServeMux(h serveHandlers) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf(`missing required query param "q"`))
			return
		}
		out, err := h.search(searchArgs{Query: q, Limit: atoiOr(r.URL.Query().Get("limit"), 0)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("GET /api/recent", func(w http.ResponseWriter, r *http.Request) {
		out, err := h.recent(recentArgs{Count: atoiOr(r.URL.Query().Get("n"), 0)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("GET /api/date/{date}", func(w http.ResponseWriter, r *http.Request) {
		out, err := h.byDate(dateArgs{Date: r.PathValue("date")})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("GET /api/range", func(w http.ResponseWriter, r *http.Request) {
		start, end := r.URL.Query().Get("start"), r.URL.Query().Get("end")
		if start == "" || end == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf(`both "start" and "end" query params are required`))
			return
		}
		out, err := h.byRange(rangeArgs{StartDate: start, EndDate: end})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("GET /api/meeting", func(w http.ResponseWriter, r *http.Request) {
		start := r.URL.Query().Get("start")
		if start == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf(`missing required query param "start"`))
			return
		}
		out, err := h.meeting(meetingArgs{
			Start:     start,
			End:       r.URL.Query().Get("end"),
			BufferMin: atoiOr(r.URL.Query().Get("buffer"), 0),
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("GET /api/get/{id}", func(w http.ResponseWriter, r *http.Request) {
		full := r.URL.Query().Get("full") == "true"
		out, err := h.getTranscript(getArgs{ID: r.PathValue("id"), Full: full})
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("GET /api/stats", func(w http.ResponseWriter, r *http.Request) {
		st, err := h.stats()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, st)
	})

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err) // embedded at build time; cannot fail at runtime
	}
	mux.Handle("/", http.FileServerFS(staticFS))

	return mux
}

// openBrowser best-effort launches the system default browser at url.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// cmdServe serves the catalog to a local browser UI, read-only. It binds
// to loopback only: this is a personal-data viewer, not a public service.
func cmdServe(cfg config, args []string) error {
	flagSet := flag.NewFlagSet("serve", flag.ExitOnError)
	port := flagSet.Int("port", 8080, "port to listen on")
	flagSet.Parse(args)

	s, err := store.OpenReadOnly(cfg.dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	mux := newServeMux(serveHandlers{s: s, loc: cfg.loc})

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	url := "http://" + addr
	fmt.Println("serving catalog at", url, "(read-only; Ctrl+C to stop)")
	if err := openBrowser(url); err != nil {
		log.Printf("could not open browser automatically: %v", err)
	}

	if err := http.Serve(ln, mux); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
