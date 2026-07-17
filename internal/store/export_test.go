package store

import "testing"

func TestExportRecords(t *testing.T) {
	s := openTemp(t)
	seed(t, s)

	all, err := s.ExportRecords("", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].ID != "a" {
		t.Errorf("all: got %d, first %v", len(all), all)
	}
	if all[0].TranscriptMD == "" || all[0].RawJSON == "" {
		t.Error("export records must carry transcript and raw json")
	}

	some, err := s.ExportRecords("2026-07-06", "2026-07-06")
	if err != nil {
		t.Fatal(err)
	}
	if len(some) != 2 {
		t.Errorf("ranged: got %d, want 2", len(some))
	}
}

func TestExportRecordsMatching(t *testing.T) {
	s := openTemp(t)
	seed(t, s) // a: "quoting pipeline", b: "kubernetes migration", c: media noise

	recs, err := s.ExportRecordsMatching("kubernetes", "", "")
	if err != nil {
		t.Fatalf("matching: %v", err)
	}
	if len(recs) != 1 || recs[0].ID != "b" {
		t.Fatalf("got %+v, want just b", recs)
	}
	if recs[0].TranscriptMD == "" {
		t.Error("matching export must carry transcripts")
	}

	// Date bounds exclude the match.
	recs, err = s.ExportRecordsMatching("kubernetes", "2026-07-01", "2026-07-05")
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 0 {
		t.Errorf("date-bounded: got %d, want 0", len(recs))
	}

	// FTS syntax characters must not error (phrase semantics).
	if _, err := s.ExportRecordsMatching(`weird "quote AND`, "", ""); err != nil {
		t.Errorf("special chars: %v", err)
	}
}
