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
