package main

import (
	"reflect"
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

func TestSpeakerRedactorNoNames(t *testing.T) {
	fr := store.FullRecord{
		Row:          store.Row{ID: "x1", Speakers: []string{"Ava", "Ben"}},
		TranscriptMD: "**Ava:** hi Ben",
		RawJSON:      `{"speakerName":"Ava"}`,
	}
	red := newSpeakerRedactor(nil)
	got := red.redact(fr)
	if got.TranscriptMD != fr.TranscriptMD || got.RawJSON != fr.RawJSON {
		t.Errorf("no-name redactor must be a no-op, got %+v", got)
	}
}

func TestSpeakerRedactorReplacesNameEverywhere(t *testing.T) {
	fr := store.FullRecord{
		Row: store.Row{ID: "x1", Speakers: []string{"Ava", "Ben"}},
		TranscriptMD: "**Ava:** hi Ben, it's ava here\n" +
			"**Ben:** hey Ava",
		RawJSON: `{"speakerName":"Ava","content":"hi Ben"}`,
	}
	red := newSpeakerRedactor([]string{"Ava"})
	got := red.redact(fr)

	if !reflect.DeepEqual(got.Speakers, []string{redactedPlaceholder, "Ben"}) {
		t.Errorf("speakers = %v, want [%s Ben]", got.Speakers, redactedPlaceholder)
	}
	if strings.Contains(got.TranscriptMD, "Ava") || strings.Contains(got.TranscriptMD, "ava") {
		t.Errorf("transcript still contains the redacted name: %q", got.TranscriptMD)
	}
	if !strings.Contains(got.TranscriptMD, redactedPlaceholder) {
		t.Errorf("transcript missing placeholder: %q", got.TranscriptMD)
	}
	if strings.Contains(got.RawJSON, "Ava") {
		t.Errorf("raw_json still contains the redacted name: %q", got.RawJSON)
	}
	// Untouched speaker's lines survive intact.
	if !strings.Contains(got.TranscriptMD, "Ben") {
		t.Errorf("unrelated speaker got clobbered: %q", got.TranscriptMD)
	}
}

func TestSpeakerRedactorWholeWordOnly(t *testing.T) {
	fr := store.FullRecord{
		Row:          store.Row{ID: "x1", Speakers: []string{"Ava"}},
		TranscriptMD: "Avalanche season and Ava's plan",
	}
	red := newSpeakerRedactor([]string{"Ava"})
	got := red.redact(fr)
	if !strings.HasPrefix(got.TranscriptMD, "Avalanche") {
		t.Errorf("word-boundary match clobbered an unrelated word: %q", got.TranscriptMD)
	}
	if !strings.Contains(got.TranscriptMD, redactedPlaceholder+"'s plan") {
		t.Errorf("expected only the standalone name redacted: %q", got.TranscriptMD)
	}
}
