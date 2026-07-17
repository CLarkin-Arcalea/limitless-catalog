package main

import (
	"path/filepath"
	"testing"
	"time"

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
