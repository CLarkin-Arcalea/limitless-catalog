package store

import (
	"testing"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/catalog"
)

func testRecord(id, updatedAt string) catalog.Record {
	return catalog.Record{
		ID:           id,
		StartUTC:     "2026-07-06T18:00:00Z",
		EndUTC:       "2026-07-06T18:30:00Z",
		LocalDate:    "2026-07-06",
		Title:        "Ben 1:1",
		DurationMin:  30,
		IsStarred:    false,
		UpdatedAt:    updatedAt,
		Speakers:     []string{"Ava", "Ben"},
		TranscriptMD: "**Ava:** hey budget talk",
		Category:     "unknown",
		RawJSON:      `{"id":"` + id + `"}`,
	}
}

func TestUpsertInsertSkipUpdate(t *testing.T) {
	s := openTemp(t)

	res, err := s.Upsert(testRecord("l1", "2026-07-06T19:00:00Z"))
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if res != Inserted {
		t.Errorf("first upsert = %v, want Inserted", res)
	}

	// Same updatedAt: skip, no rewrite.
	res, err = s.Upsert(testRecord("l1", "2026-07-06T19:00:00Z"))
	if err != nil {
		t.Fatalf("skip: %v", err)
	}
	if res != Skipped {
		t.Errorf("second upsert = %v, want Skipped", res)
	}

	// Changed updatedAt: rewrite.
	r := testRecord("l1", "2026-07-06T20:00:00Z")
	r.Title = "Ben 1:1 (revised)"
	r.TranscriptMD = "**Ava:** revised transcript"
	res, err = s.Upsert(r)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if res != Updated {
		t.Errorf("third upsert = %v, want Updated", res)
	}

	var title string
	if err := s.DB().QueryRow(`SELECT title FROM lifelogs WHERE id='l1'`).Scan(&title); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if title != "Ben 1:1 (revised)" {
		t.Errorf("title = %q", title)
	}

	var n int
	if err := s.DB().QueryRow(`SELECT COUNT(*) FROM lifelogs`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("row count = %d, want 1", n)
	}
}

func TestUpsertSyncsFTS(t *testing.T) {
	s := openTemp(t)
	if _, err := s.Upsert(testRecord("l1", "2026-07-06T19:00:00Z")); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := s.DB().QueryRow(
		`SELECT COUNT(*) FROM lifelogs_fts WHERE lifelogs_fts MATCH '"budget"'`).Scan(&n); err != nil {
		t.Fatalf("fts query: %v", err)
	}
	if n != 1 {
		t.Errorf("fts matches = %d, want 1", n)
	}

	// Rewrite must not leave a stale FTS row behind.
	r := testRecord("l1", "2026-07-06T20:00:00Z")
	r.TranscriptMD = "**Ava:** now about kubernetes"
	if _, err := s.Upsert(r); err != nil {
		t.Fatal(err)
	}
	if err := s.DB().QueryRow(
		`SELECT COUNT(*) FROM lifelogs_fts WHERE lifelogs_fts MATCH '"budget"'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("stale fts rows = %d, want 0", n)
	}
	if err := s.DB().QueryRow(
		`SELECT COUNT(*) FROM lifelogs_fts WHERE lifelogs_fts MATCH '"kubernetes"'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("new fts rows = %d, want 1", n)
	}
}
