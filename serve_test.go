package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

func serveHandlersForTest(t *testing.T) serveHandlers {
	t.Helper()
	path := seedMain(t)
	ro, err := store.OpenReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ro.Close() })
	return serveHandlers{s: ro, loc: time.UTC}
}

func TestServeSearchHandler(t *testing.T) {
	h := serveHandlersForTest(t)
	out, err := h.search(searchArgs{Query: "budget"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Results) != 1 || out.Results[0].ID != "m1" {
		t.Errorf("got %+v", out)
	}
}

func TestServeMeetingHandler(t *testing.T) {
	h := serveHandlersForTest(t)
	out, err := h.meeting(meetingArgs{Start: "2026-07-06 18:15", End: "2026-07-06 18:40"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Results) != 1 || out.Results[0].ID != "m2" {
		t.Errorf("got %+v", out)
	}
	if _, err := h.meeting(meetingArgs{Start: "not a time"}); err == nil {
		t.Error("want error for bad datetime")
	}
}

func TestServeListAndRecentHandlers(t *testing.T) {
	h := serveHandlersForTest(t)
	if out, err := h.byDate(dateArgs{Date: "2026-07-05"}); err != nil || len(out.Results) != 1 {
		t.Errorf("byDate: %v %+v", err, out)
	}
	if out, err := h.byRange(rangeArgs{StartDate: "2026-07-05", EndDate: "2026-07-06"}); err != nil || len(out.Results) != 2 {
		t.Errorf("byRange: %v %+v", err, out)
	}
	if out, err := h.recent(recentArgs{Count: 1}); err != nil || len(out.Results) != 1 || out.Results[0].ID != "m2" {
		t.Errorf("recent: %v %+v", err, out)
	}
}

func TestServeGetTranscriptHandler(t *testing.T) {
	h := serveHandlersForTest(t)
	out, err := h.getTranscript(getArgs{ID: "m1", Full: true})
	if err != nil {
		t.Fatal(err)
	}
	if out.Title != "Budget review" || out.TranscriptMD == "" {
		t.Errorf("got %+v", out)
	}
	meta, err := h.getTranscript(getArgs{ID: "m1", Full: false})
	if err != nil {
		t.Fatal(err)
	}
	if meta.TranscriptMD != "" {
		t.Error("full=false must omit transcript text")
	}
	if _, err := h.getTranscript(getArgs{ID: "nope"}); err == nil {
		t.Error("want error for unknown id")
	}
}

func TestServeStatsHandler(t *testing.T) {
	h := serveHandlersForTest(t)
	out, err := h.stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if out.Total != 2 || out.FirstDate != "2026-07-05" {
		t.Errorf("stats = %+v", out)
	}
}

func TestMuxRoutes(t *testing.T) {
	h := serveHandlersForTest(t)
	srv := httptest.NewServer(newServeMux(h))
	defer srv.Close()

	get := func(path string) (*http.Response, map[string]any) {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		return resp, body
	}

	if resp, body := get("/api/search?q=budget"); resp.StatusCode != http.StatusOK {
		t.Errorf("search status = %d, body = %v", resp.StatusCode, body)
	} else if results, _ := body["results"].([]any); len(results) != 1 {
		t.Errorf("search results = %v", body)
	}

	if resp, body := get("/api/search"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("search without q: status = %d, body = %v", resp.StatusCode, body)
	}

	if resp, body := get("/api/recent?n=1"); resp.StatusCode != http.StatusOK {
		t.Errorf("recent status = %d, body = %v", resp.StatusCode, body)
	} else if results, _ := body["results"].([]any); len(results) != 1 {
		t.Errorf("recent results = %v", body)
	}

	if resp, body := get("/api/date/2026-07-05"); resp.StatusCode != http.StatusOK {
		t.Errorf("date status = %d, body = %v", resp.StatusCode, body)
	} else if results, _ := body["results"].([]any); len(results) != 1 {
		t.Errorf("date results = %v", body)
	}

	if resp, body := get("/api/range?start=2026-07-05&end=2026-07-06"); resp.StatusCode != http.StatusOK {
		t.Errorf("range status = %d, body = %v", resp.StatusCode, body)
	} else if results, _ := body["results"].([]any); len(results) != 2 {
		t.Errorf("range results = %v", body)
	}

	if resp, body := get("/api/range?start=2026-07-05"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("range missing end: status = %d, body = %v", resp.StatusCode, body)
	}

	if resp, body := get("/api/meeting?start=" + "2026-07-06+18:15" + "&end=" + "2026-07-06+18:40"); resp.StatusCode != http.StatusOK {
		t.Errorf("meeting status = %d, body = %v", resp.StatusCode, body)
	} else if results, _ := body["results"].([]any); len(results) != 1 {
		t.Errorf("meeting results = %v", body)
	}

	if resp, body := get("/api/meeting?start=not-a-time"); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("meeting bad time: status = %d, body = %v", resp.StatusCode, body)
	}

	if resp, body := get("/api/get/m1?full=true"); resp.StatusCode != http.StatusOK {
		t.Errorf("get status = %d, body = %v", resp.StatusCode, body)
	} else if body["transcript_md"] == "" || body["transcript_md"] == nil {
		t.Errorf("get full=true missing transcript: %v", body)
	}

	if resp, body := get("/api/get/nope"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("get unknown id: status = %d, body = %v", resp.StatusCode, body)
	}

	if resp, body := get("/api/stats"); resp.StatusCode != http.StatusOK {
		t.Errorf("stats status = %d, body = %v", resp.StatusCode, body)
	} else if body["total"] != float64(2) {
		t.Errorf("stats total = %v", body)
	}

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("index status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("index content-type = %q", ct)
	}
}
