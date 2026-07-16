package store

import (
	"testing"
)

func TestStats(t *testing.T) {
	s := openTemp(t)
	seed(t, s) // a on 07-05, b and c on 07-06; c is media
	if err := s.SetState("last_ingest_run", "2026-07-07T01:00:00Z"); err != nil {
		t.Fatal(err)
	}

	st, err := s.Stats("")
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if st.Total != 3 {
		t.Errorf("Total = %d", st.Total)
	}
	if st.FirstDate != "2026-07-05" || st.LastDate != "2026-07-06" {
		t.Errorf("dates = %s..%s", st.FirstDate, st.LastDate)
	}
	// 30 + 45 + 10 minutes = 85m = 1.4166h
	if st.TotalHours < 1.41 || st.TotalHours > 1.42 {
		t.Errorf("TotalHours = %v", st.TotalHours)
	}
	if len(st.PerMonth) != 1 || st.PerMonth[0].Month != "2026-07" || st.PerMonth[0].Count != 3 {
		t.Errorf("PerMonth = %+v", st.PerMonth)
	}
	if len(st.EmptyDays) != 0 {
		t.Errorf("EmptyDays = %v, want none (both days covered)", st.EmptyDays)
	}
	if st.ByCategory["media"] != 1 || st.ByCategory["unknown"] != 2 {
		t.Errorf("ByCategory = %v", st.ByCategory)
	}
	if st.LastIngest != "2026-07-07T01:00:00Z" {
		t.Errorf("LastIngest = %q", st.LastIngest)
	}
}

func TestStatsFindsGapDays(t *testing.T) {
	s := openTemp(t)
	// Two logs three days apart: 07-02 and 07-05 leave 07-03, 07-04 empty.
	for _, r := range []struct{ id, date, start, end string }{
		{"g1", "2026-07-02", "2026-07-02T10:00:00Z", "2026-07-02T10:30:00Z"},
		{"g2", "2026-07-05", "2026-07-05T10:00:00Z", "2026-07-05T10:30:00Z"},
	} {
		rec := testRecord(r.id, "u")
		rec.LocalDate = r.date
		rec.StartUTC = r.start
		rec.EndUTC = r.end
		if _, err := s.Upsert(rec); err != nil {
			t.Fatal(err)
		}
	}
	st, err := s.Stats("")
	if err != nil {
		t.Fatal(err)
	}
	if len(st.EmptyDays) != 2 || st.EmptyDays[0] != "2026-07-03" || st.EmptyDays[1] != "2026-07-04" {
		t.Errorf("EmptyDays = %v", st.EmptyDays)
	}
}

func TestStatsEmptyCatalog(t *testing.T) {
	s := openTemp(t)
	st, err := s.Stats("")
	if err != nil {
		t.Fatal(err)
	}
	if st.Total != 0 || st.FirstDate != "" || len(st.EmptyDays) != 0 {
		t.Errorf("empty stats = %+v", st)
	}
}
