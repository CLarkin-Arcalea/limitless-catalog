package main

import (
	"strings"
	"testing"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Ben 1:1":                  "ben-1-1",
		"  Weird -- Chars!! (ok) ": "weird-chars-ok",
		"":                         "untitled",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
	long := strings.Repeat("word ", 30)
	if got := slugify(long); len(got) > 60 {
		t.Errorf("slug too long: %d chars", len(got))
	}
}

func TestExportFilename(t *testing.T) {
	chicago, _ := time.LoadLocation("America/Chicago")
	fr := store.FullRecord{Row: store.Row{
		ID: "abcdef1234567890", Title: "Ben 1:1",
		StartUTC: "2026-07-06T18:00:00Z", LocalDate: "2026-07-06"}}
	got := exportFilename(fr, chicago)
	// 18:00 UTC = 13:00 Chicago
	want := "2026-07-06-1300-ben-1-1-abcdef12.md"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderExportMarkdown(t *testing.T) {
	fr := store.FullRecord{
		Row: store.Row{ID: "x1", Title: "Ben 1:1", LocalDate: "2026-07-06",
			StartUTC: "2026-07-06T18:00:00Z", EndUTC: "2026-07-06T18:45:00Z",
			Speakers: []string{"Ava", "Ben"}, Category: "unknown"},
		TranscriptMD: "**Ava:** hello",
	}
	out := renderExportMarkdown(fr)
	for _, want := range []string{"title: Ben 1:1", "id: x1",
		"speakers: Ava, Ben", "**Ava:** hello"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
