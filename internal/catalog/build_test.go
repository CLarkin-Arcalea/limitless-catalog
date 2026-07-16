package catalog

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/limitless"
)

func speakerLog() limitless.Lifelog {
	return limitless.Lifelog{
		ID: "log1", Title: "Ben 1:1",
		StartTime: "2026-07-06T18:00:00.000Z",
		EndTime:   "2026-07-06T18:30:00.000Z",
		UpdatedAt: "2026-07-06T19:00:00.000Z",
		IsStarred: true,
		Contents: []limitless.ContentNode{
			{Type: "heading1", Content: "Ben 1:1", Children: []limitless.ContentNode{
				{Type: "blockquote", Content: "hey", SpeakerName: "Ava", SpeakerIdentifier: "user"},
				{Type: "blockquote", Content: "hello", SpeakerName: "Ben"},
				{Type: "blockquote", Content: "again", SpeakerName: "Ben"},
			}},
		},
	}
}

func TestExtractSpeakers(t *testing.T) {
	got := ExtractSpeakers(speakerLog())
	want := []string{"Ava", "Ben"} // distinct, sorted
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if !HasUserSpeaker(speakerLog()) {
		t.Error("HasUserSpeaker should be true")
	}
}

func TestCategorize(t *testing.T) {
	media := "Don't forget to subscribe to the channel. This episode is sponsored by Acme."
	cases := []struct {
		name       string
		transcript string
		hasUser    bool
		want       string
	}{
		{"user present is never media", media, true, "unknown"},
		{"two media markers, no user", media, false, "media"},
		{"one marker is not enough", "please subscribe", false, "unknown"},
		{"plain conversation", "let's review the budget", false, "unknown"},
	}
	for _, c := range cases {
		if got := Categorize(c.transcript, c.hasUser); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestBuild(t *testing.T) {
	chicago, _ := time.LoadLocation("America/Chicago")
	r, err := Build(speakerLog(), chicago)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if r.ID != "log1" || r.Title != "Ben 1:1" || !r.IsStarred {
		t.Errorf("basic fields wrong: %+v", r)
	}
	if r.StartUTC != "2026-07-06T18:00:00Z" || r.EndUTC != "2026-07-06T18:30:00Z" {
		t.Errorf("times not normalized: %s / %s", r.StartUTC, r.EndUTC)
	}
	if r.LocalDate != "2026-07-06" { // 18:00 UTC = 13:00 Chicago
		t.Errorf("LocalDate = %q", r.LocalDate)
	}
	if r.DurationMin != 30 {
		t.Errorf("DurationMin = %v", r.DurationMin)
	}
	if r.UpdatedAt != "2026-07-06T19:00:00Z" {
		t.Errorf("UpdatedAt = %q", r.UpdatedAt)
	}
	if !reflect.DeepEqual(r.Speakers, []string{"Ava", "Ben"}) {
		t.Errorf("Speakers = %v", r.Speakers)
	}
	if !strings.Contains(r.TranscriptMD, "**Ben:** hello") {
		t.Errorf("transcript missing content: %q", r.TranscriptMD)
	}
	if r.Category != "unknown" { // user speaker present
		t.Errorf("Category = %q", r.Category)
	}
	if !strings.Contains(r.RawJSON, `"id":"log1"`) {
		t.Errorf("RawJSON missing: %q", r.RawJSON)
	}
}

func TestBuildRejectsBadTimes(t *testing.T) {
	l := speakerLog()
	l.StartTime = "garbage"
	if _, err := Build(l, time.UTC); err == nil {
		t.Fatal("want error for unparseable startTime")
	}
}
