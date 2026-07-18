package main

import (
	"testing"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

func TestScanForPII(t *testing.T) {
	cases := []struct {
		name string
		text string
		want string // "" means no finding
	}{
		{"ssn", "my SSN is 123-45-6789 ok", "ssn"},
		{"credit card spaced", "card 4111 2222 3333 4444 expires", "credit_card"},
		{"credit card dashed", "card 4111-2222-3333-4444 expires", "credit_card"},
		{"email", "reach me at chris@example.com please", "email"},
		{"phone dashed", "call 555-123-4567 anytime", "phone"},
		{"phone parens", "call (555) 123-4567 anytime", "phone"},
		{"clean", "just a normal conversation about kubernetes", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fr := store.FullRecord{Row: store.Row{ID: "id1", LocalDate: "2026-07-06"}, TranscriptMD: tc.text}
			findings := scanForPII(fr)
			if tc.want == "" {
				if len(findings) != 0 {
					t.Errorf("expected no findings, got %+v", findings)
				}
				return
			}
			found := false
			for _, f := range findings {
				if f.Kind == tc.want {
					found = true
					if f.LifelogID != "id1" || f.LocalDate != "2026-07-06" {
						t.Errorf("finding location wrong: %+v", f)
					}
				}
			}
			if !found {
				t.Errorf("expected a %q finding in %+v", tc.want, findings)
			}
		})
	}
}

func TestScanForPIIMultipleKindsOneRecord(t *testing.T) {
	fr := store.FullRecord{
		Row:          store.Row{ID: "id1", LocalDate: "2026-07-06"},
		TranscriptMD: "ssn 123-45-6789, email chris@example.com, phone 555-123-4567",
	}
	findings := scanForPII(fr)
	kinds := map[string]bool{}
	for _, f := range findings {
		kinds[f.Kind] = true
	}
	for _, want := range []string{"ssn", "email", "phone"} {
		if !kinds[want] {
			t.Errorf("missing kind %q in %+v", want, findings)
		}
	}
}
