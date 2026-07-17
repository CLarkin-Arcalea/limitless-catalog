package main

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

// mcpHandlers binds the read-only store to tool implementations so each
// handler is a plain method that tests can call directly.
type mcpHandlers struct {
	s      *store.Store
	loc    *time.Location
	dbPath string
}

func (h mcpHandlers) stats() (store.Stats, error) {
	return h.s.Stats(h.dbPath)
}

type emptyArgs struct{}

type searchArgs struct {
	Query string `json:"query" jsonschema:"full-text phrase to search for in titles and transcripts"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum results, default 20"`
}
type meetingArgs struct {
	Start     string `json:"start" jsonschema:"meeting start as YYYY-MM-DD HH:MM in the server's timezone"`
	End       string `json:"end,omitempty" jsonschema:"meeting end, default start plus one hour"`
	BufferMin int    `json:"buffer_min,omitempty" jsonschema:"minutes of slack on each side, default 10"`
}
type dateArgs struct {
	Date string `json:"date" jsonschema:"local date YYYY-MM-DD"`
}
type rangeArgs struct {
	StartDate string `json:"start_date" jsonschema:"first local date YYYY-MM-DD inclusive"`
	EndDate   string `json:"end_date" jsonschema:"last local date YYYY-MM-DD inclusive"`
}
type recentArgs struct {
	Count int `json:"count,omitempty" jsonschema:"how many newest conversations, default 10"`
}
type getArgs struct {
	ID   string `json:"id" jsonschema:"lifelog id from a listing or search result"`
	Full bool   `json:"full,omitempty" jsonschema:"include the full transcript text"`
}

type rowsOut struct {
	Results []store.Row `json:"results"`
}
type recordOut struct {
	store.FullRecord
}

func (h mcpHandlers) search(a searchArgs) (rowsOut, error) {
	limit := a.Limit
	if limit <= 0 {
		limit = 20
	}
	rows, err := h.s.Search(a.Query, limit)
	return rowsOut{Results: rows}, err
}

func (h mcpHandlers) meeting(a meetingArgs) (rowsOut, error) {
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

func (h mcpHandlers) byDate(a dateArgs) (rowsOut, error) {
	rows, err := h.s.ByDate(a.Date)
	return rowsOut{Results: rows}, err
}

func (h mcpHandlers) byRange(a rangeArgs) (rowsOut, error) {
	rows, err := h.s.ByRange(a.StartDate, a.EndDate)
	return rowsOut{Results: rows}, err
}

func (h mcpHandlers) recent(a recentArgs) (rowsOut, error) {
	n := a.Count
	if n <= 0 {
		n = 10
	}
	rows, err := h.s.Recent(n)
	return rowsOut{Results: rows}, err
}

func (h mcpHandlers) getTranscript(a getArgs) (recordOut, error) {
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

// newMCPServer registers every read-only tool on a fresh MCP server.
func newMCPServer(s *store.Store, loc *time.Location, dbPath string) *mcp.Server {
	h := mcpHandlers{s: s, loc: loc, dbPath: dbPath}
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "limitless-catalog",
		Version: "1.1.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "catalog_stats",
		Description: "Catalog coverage: totals, date range, hours, per-month counts, gap days, categories.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args emptyArgs) (*mcp.CallToolResult, store.Stats, error) {
		st, err := h.stats()
		if err != nil {
			return nil, store.Stats{}, err
		}
		return nil, st, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_transcripts",
		Description: "Full-text search over conversation titles and transcripts. Returns metadata and snippets, not full text; follow up with get_transcript.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, a searchArgs) (*mcp.CallToolResult, rowsOut, error) {
		out, err := h.search(a)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_meeting",
		Description: "Find conversations overlapping a time window (calendar-meeting lookup). Overlap, not containment.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, a meetingArgs) (*mcp.CallToolResult, rowsOut, error) {
		out, err := h.meeting(a)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_by_date",
		Description: "List conversations on one local date.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, a dateArgs) (*mcp.CallToolResult, rowsOut, error) {
		out, err := h.byDate(a)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_range",
		Description: "List conversations across an inclusive local-date range.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, a rangeArgs) (*mcp.CallToolResult, rowsOut, error) {
		out, err := h.byRange(a)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "recent",
		Description: "List the newest conversations.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, a recentArgs) (*mcp.CallToolResult, rowsOut, error) {
		out, err := h.recent(a)
		return nil, out, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_transcript",
		Description: "Fetch one conversation by id; set full=true for the complete transcript markdown.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, a getArgs) (*mcp.CallToolResult, recordOut, error) {
		out, err := h.getTranscript(a)
		return nil, out, err
	})

	return server
}

// cmdMCP serves the catalog to MCP clients over stdio, read-only.
func cmdMCP(cfg config, args []string) error {
	s, err := store.OpenReadOnly(cfg.dbPath)
	if err != nil {
		return err
	}
	defer s.Close()
	server := newMCPServer(s, cfg.loc, cfg.dbPath)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("mcp server: %w", err)
	}
	return nil
}
