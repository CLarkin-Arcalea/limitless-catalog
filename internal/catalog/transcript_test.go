package catalog

import (
	"testing"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/limitless"
)

func TestAssembleTranscriptPrefersMarkdown(t *testing.T) {
	l := limitless.Lifelog{
		Markdown: "# Provided\n\nAlready assembled.",
		Contents: []limitless.ContentNode{{Type: "heading1", Content: "Ignored"}},
	}
	if got := AssembleTranscript(l); got != "# Provided\n\nAlready assembled." {
		t.Errorf("got %q", got)
	}
}

func TestAssembleTranscriptBuildsFromContents(t *testing.T) {
	l := limitless.Lifelog{
		Markdown: "   ", // whitespace only: treat as absent
		Contents: []limitless.ContentNode{
			{
				Type: "heading1", Content: "Standup",
				Children: []limitless.ContentNode{
					{Type: "blockquote", Content: "morning all",
						SpeakerName: "Ava", SpeakerIdentifier: "user"},
					{Type: "blockquote", Content: "hi", SpeakerName: ""},
				},
			},
			{Type: "heading2", Content: "Decisions"},
			{Type: "heading3", Content: "Budget"},
		},
	}
	want := "# Standup\n\n" +
		"**Ava:** morning all\n\n" +
		"**Unknown:** hi\n\n" +
		"## Decisions\n\n" +
		"### Budget"
	if got := AssembleTranscript(l); got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestAssembleTranscriptEmpty(t *testing.T) {
	if got := AssembleTranscript(limitless.Lifelog{}); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}
