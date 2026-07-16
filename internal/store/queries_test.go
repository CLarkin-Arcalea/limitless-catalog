package store

import (
	"testing"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/catalog"
)

// seed inserts three lifelogs across two days.
func seed(t *testing.T, s *Store) {
	t.Helper()
	recs := []catalog.Record{
		{ID: "a", StartUTC: "2026-07-05T14:00:00Z", EndUTC: "2026-07-05T14:30:00Z",
			LocalDate: "2026-07-05", Title: "AMS quoting call", DurationMin: 30,
			UpdatedAt: "u1", Speakers: []string{"Ava", "Mike"},
			TranscriptMD: "**Ava:** quoting pipeline discussion", Category: "unknown",
			RawJSON: `{"id":"a"}`},
		{ID: "b", StartUTC: "2026-07-06T18:00:00Z", EndUTC: "2026-07-06T18:45:00Z",
			LocalDate: "2026-07-06", Title: "Ben 1:1", DurationMin: 45,
			UpdatedAt: "u2", Speakers: []string{"Ava", "Ben"},
			TranscriptMD: "**Ben:** kubernetes migration status", Category: "unknown",
			RawJSON: `{"id":"b"}`},
		{ID: "c", StartUTC: "2026-07-06T20:00:00Z", EndUTC: "2026-07-06T20:10:00Z",
			LocalDate: "2026-07-06", Title: "Radio noise", DurationMin: 10,
			UpdatedAt: "u3", Speakers: nil,
			TranscriptMD: "subscribe now, sponsored by", Category: "media",
			RawJSON: `{"id":"c"}`},
	}
	for _, r := range recs {
		if _, err := s.Upsert(r); err != nil {
			t.Fatalf("seed %s: %v", r.ID, err)
		}
	}
}

func TestSearch(t *testing.T) {
	s := openTemp(t)
	seed(t, s)

	rows, err := s.Search("kubernetes", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "b" {
		t.Fatalf("got %+v, want just b", rows)
	}
	if rows[0].Snippet == "" {
		t.Error("want non-empty snippet")
	}
	if rows[0].Speakers[1] != "Ben" {
		t.Errorf("speakers not decoded: %v", rows[0].Speakers)
	}

	// A term with FTS syntax characters must not error (phrase-quoted).
	if _, err := s.Search(`kubernetes AND "weird`, 10); err != nil {
		t.Errorf("special chars: %v", err)
	}
}

func TestByDateAndRange(t *testing.T) {
	s := openTemp(t)
	seed(t, s)

	rows, err := s.ByDate("2026-07-06")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].ID != "b" || rows[1].ID != "c" {
		t.Errorf("ByDate got %+v", rows)
	}

	rows, err = s.ByRange("2026-07-05", "2026-07-06")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].ID != "a" {
		t.Errorf("ByRange got %d rows, first %v", len(rows), rows)
	}
}

func TestRecent(t *testing.T) {
	s := openTemp(t)
	seed(t, s)

	rows, err := s.Recent(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].ID != "c" || rows[1].ID != "b" {
		t.Errorf("Recent got %+v", rows)
	}
}

func TestMeetingOverlap(t *testing.T) {
	s := openTemp(t)
	seed(t, s)

	// Calendar window 18:15-18:40 UTC overlaps only "b" (18:00-18:45).
	start, _ := time.Parse(time.RFC3339, "2026-07-06T18:15:00Z")
	end, _ := time.Parse(time.RFC3339, "2026-07-06T18:40:00Z")
	rows, err := s.Meeting(start, end, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != "b" {
		t.Errorf("Meeting got %+v", rows)
	}

	// 19:50 meeting with 15m buffer catches "c" (starts 20:00).
	start2, _ := time.Parse(time.RFC3339, "2026-07-06T19:30:00Z")
	end2, _ := time.Parse(time.RFC3339, "2026-07-06T19:50:00Z")
	rows, err = s.Meeting(start2, end2, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != "c" {
		t.Errorf("buffered Meeting got %+v", rows)
	}
}

func TestGet(t *testing.T) {
	s := openTemp(t)
	seed(t, s)

	fr, err := s.Get("b")
	if err != nil {
		t.Fatal(err)
	}
	if fr == nil || fr.Title != "Ben 1:1" || fr.TranscriptMD == "" || fr.IngestedAt == "" {
		t.Errorf("Get got %+v", fr)
	}

	missing, err := s.Get("nope")
	if err != nil {
		t.Fatal(err)
	}
	if missing != nil {
		t.Errorf("want nil for missing id, got %+v", missing)
	}
}

func TestStateAndMaxLocalDate(t *testing.T) {
	s := openTemp(t)

	if v, err := s.GetState("k"); err != nil || v != "" {
		t.Errorf("empty state: v=%q err=%v", v, err)
	}
	if err := s.SetState("k", "v1"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetState("k", "v2"); err != nil { // overwrite
		t.Fatal(err)
	}
	if v, _ := s.GetState("k"); v != "v2" {
		t.Errorf("state = %q, want v2", v)
	}

	if d, err := s.MaxLocalDate(); err != nil || d != "" {
		t.Errorf("empty MaxLocalDate: %q %v", d, err)
	}
	seed(t, s)
	if d, _ := s.MaxLocalDate(); d != "2026-07-06" {
		t.Errorf("MaxLocalDate = %q", d)
	}
}
