package catalog

import (
	"testing"
	"time"
)

func TestNormalizeUTC(t *testing.T) {
	cases := []struct {
		in      string
		wantStr string
		wantErr bool
	}{
		{"2026-07-06T14:00:00.000Z", "2026-07-06T14:00:00Z", false},
		{"2026-07-06T09:00:00-05:00", "2026-07-06T14:00:00Z", false},
		{"2026-07-06T14:00:00Z", "2026-07-06T14:00:00Z", false},
		{"not-a-time", "", true},
		{"", "", true},
	}
	for _, c := range cases {
		got, gotT, err := NormalizeUTC(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("NormalizeUTC(%q): want error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizeUTC(%q): %v", c.in, err)
			continue
		}
		if got != c.wantStr {
			t.Errorf("NormalizeUTC(%q) = %q, want %q", c.in, got, c.wantStr)
		}
		if gotT.Location() != time.UTC {
			t.Errorf("NormalizeUTC(%q) time not UTC", c.in)
		}
	}
}

func TestLocalDateAndDuration(t *testing.T) {
	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	// 2026-07-07 02:30 UTC is still 2026-07-06 21:30 in Chicago (CDT, UTC-5).
	start, _ := time.Parse(time.RFC3339, "2026-07-07T02:30:00Z")
	end, _ := time.Parse(time.RFC3339, "2026-07-07T03:15:00Z")

	if got := LocalDate(start, chicago); got != "2026-07-06" {
		t.Errorf("LocalDate = %q, want 2026-07-06", got)
	}
	if got := DurationMinutes(start, end); got != 45.0 {
		t.Errorf("DurationMinutes = %v, want 45", got)
	}
}
