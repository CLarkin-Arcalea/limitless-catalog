package store

import "testing"

func TestSpeakers(t *testing.T) {
	s := openTemp(t)
	seed(t, s) // a: Ava,Mike on 07-05; b: Ava,Ben on 07-06; c: none on 07-06

	stats, err := s.Speakers()
	if err != nil {
		t.Fatalf("Speakers: %v", err)
	}
	if len(stats) != 3 {
		t.Fatalf("got %d speakers, want 3: %+v", len(stats), stats)
	}

	// Ava appears twice (a, b); ranked first.
	if stats[0].Name != "Ava" || stats[0].Count != 2 {
		t.Errorf("stats[0] = %+v, want Ava count 2", stats[0])
	}
	if stats[0].FirstSeen != "2026-07-05" || stats[0].LastSeen != "2026-07-06" {
		t.Errorf("Ava first/last = %s/%s", stats[0].FirstSeen, stats[0].LastSeen)
	}
	if stats[0].DaysSinceLast != 0 {
		t.Errorf("Ava DaysSinceLast = %d, want 0 (last lifelog is also 07-06)", stats[0].DaysSinceLast)
	}
	if stats[0].LongestGapDays != 1 {
		t.Errorf("Ava LongestGapDays = %d, want 1 (07-05 to 07-06)", stats[0].LongestGapDays)
	}

	// Ben and Mike each appear once; tie-broken alphabetically.
	if stats[1].Name != "Ben" || stats[1].Count != 1 {
		t.Errorf("stats[1] = %+v, want Ben count 1", stats[1])
	}
	if stats[1].DaysSinceLast != 0 {
		t.Errorf("Ben DaysSinceLast = %d, want 0", stats[1].DaysSinceLast)
	}
	if stats[2].Name != "Mike" || stats[2].Count != 1 {
		t.Errorf("stats[2] = %+v, want Mike count 1", stats[2])
	}
	if stats[2].DaysSinceLast != 1 {
		t.Errorf("Mike DaysSinceLast = %d, want 1 (last seen 07-05, catalog max 07-06)", stats[2].DaysSinceLast)
	}
}

func TestSpeakerSingleLookup(t *testing.T) {
	s := openTemp(t)
	seed(t, s)

	st, err := s.Speaker("Ava")
	if err != nil {
		t.Fatalf("Speaker: %v", err)
	}
	if st == nil || st.Count != 2 {
		t.Errorf("Speaker(Ava) = %+v", st)
	}

	missing, err := s.Speaker("Nobody")
	if err != nil {
		t.Fatal(err)
	}
	if missing != nil {
		t.Errorf("want nil for unknown speaker, got %+v", missing)
	}
}

func TestSpeakersEmptyCatalog(t *testing.T) {
	s := openTemp(t)
	stats, err := s.Speakers()
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 0 {
		t.Errorf("got %+v, want none", stats)
	}
}

func TestLongestGapDaysMultipleAppearances(t *testing.T) {
	s := openTemp(t)
	for _, r := range []struct{ id, date string }{
		{"g1", "2026-01-01"},
		{"g2", "2026-01-03"}, // gap 2
		{"g3", "2026-01-20"}, // gap 17
	} {
		rec := testRecord(r.id, "u-"+r.id)
		rec.LocalDate = r.date
		rec.StartUTC = r.date + "T10:00:00Z"
		rec.EndUTC = r.date + "T10:30:00Z"
		rec.Speakers = []string{"Gina"}
		if _, err := s.Upsert(rec); err != nil {
			t.Fatal(err)
		}
	}
	st, err := s.Speaker("Gina")
	if err != nil {
		t.Fatal(err)
	}
	if st == nil || st.LongestGapDays != 17 {
		t.Errorf("Gina = %+v, want LongestGapDays 17", st)
	}
	if st.Count != 3 {
		t.Errorf("Gina Count = %d, want 3", st.Count)
	}
}
