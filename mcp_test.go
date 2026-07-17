package main

import (
	"context"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/catalog"
	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

// seedMain writes two records into a temp catalog and returns its path.
func seedMain(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mcp.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, r := range []catalog.Record{
		{ID: "m1", StartUTC: "2026-07-05T14:00:00Z", EndUTC: "2026-07-05T14:30:00Z",
			LocalDate: "2026-07-05", Title: "Budget review", DurationMin: 30,
			UpdatedAt: "u1", Speakers: []string{"Ava", "Ben"},
			TranscriptMD: "**Ava:** quarterly budget numbers", Category: "unknown",
			RawJSON: `{"id":"m1"}`},
		{ID: "m2", StartUTC: "2026-07-06T18:00:00Z", EndUTC: "2026-07-06T18:45:00Z",
			LocalDate: "2026-07-06", Title: "Planning sync", DurationMin: 45,
			UpdatedAt: "u2", Speakers: []string{"Ava"},
			TranscriptMD: "**Ava:** roadmap milestones ahead", Category: "unknown",
			RawJSON: `{"id":"m2"}`},
	} {
		if _, err := s.Upsert(r); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func handlersForTest(t *testing.T) mcpHandlers {
	t.Helper()
	path := seedMain(t)
	ro, err := store.OpenReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ro.Close() })
	return mcpHandlers{s: ro, loc: time.UTC, dbPath: path}
}

func TestSearchHandler(t *testing.T) {
	h := handlersForTest(t)
	out, err := h.search(searchArgs{Query: "budget"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Results) != 1 || out.Results[0].ID != "m1" {
		t.Errorf("got %+v", out)
	}
	if out.Results[0].Snippet == "" {
		t.Error("want snippet")
	}
}

func TestMeetingHandler(t *testing.T) {
	h := handlersForTest(t)
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

func TestListAndRecentHandlers(t *testing.T) {
	h := handlersForTest(t)
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

func TestGetTranscriptHandler(t *testing.T) {
	h := handlersForTest(t)
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

func TestNewMCPServerRegistersAllTools(t *testing.T) {
	path := seedMain(t)
	ro, err := store.OpenReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	// Construction itself is half the test: a bad jsonschema tag or an
	// unreflectable In/Out type panics inside mcp.AddTool.
	server := newMCPServer(ro, time.UTC, path)
	if server == nil {
		t.Fatal("newMCPServer returned nil")
	}

	// Roundtrip over an in-memory transport and list the registered tools.
	ctx := context.Background()
	serverT, clientT := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer ss.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	var got []string
	for _, tool := range res.Tools {
		got = append(got, tool.Name)
	}
	sort.Strings(got)
	want := []string{"catalog_stats", "find_meeting", "get_transcript",
		"list_by_date", "list_range", "recent", "search_transcripts"}
	if len(got) != len(want) {
		t.Fatalf("registered tools = %v, want %v", got, want)
	}
	for i, name := range want {
		if got[i] != name {
			t.Errorf("registered tools = %v, want %v", got, want)
			break
		}
	}
}

func TestStatsHandler(t *testing.T) {
	path := seedMain(t)
	ro, err := store.OpenReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()

	h := mcpHandlers{s: ro, loc: time.UTC, dbPath: path}
	out, err := h.stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if out.Total != 2 || out.FirstDate != "2026-07-05" {
		t.Errorf("stats = %+v", out)
	}
}
