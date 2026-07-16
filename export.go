package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

// slugify lowercases s and reduces it to hyphen-separated alphanumerics,
// capped at 60 chars. Empty input becomes "untitled".
func slugify(s string) string {
	var b strings.Builder
	lastHyphen := true // suppress leading hyphen
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		default:
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "untitled"
	}
	if len(out) > 60 {
		out = strings.Trim(out[:60], "-")
	}
	return out
}

// exportFilename builds YYYY-MM-DD-HHMM-<slug>-<id8>.md using the local
// wall-clock time of the recording.
func exportFilename(fr store.FullRecord, loc *time.Location) string {
	hhmm := "0000"
	if t, err := time.Parse(time.RFC3339, fr.StartUTC); err == nil {
		hhmm = t.In(loc).Format("1504")
	}
	id := fr.ID
	if len(id) > 8 {
		id = id[:8]
	}
	return fmt.Sprintf("%s-%s-%s-%s.md", fr.LocalDate, hhmm, slugify(fr.Title), id)
}

// renderExportMarkdown renders one record as a standalone markdown file
// with a YAML-ish metadata header.
func renderExportMarkdown(fr store.FullRecord) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "title: %s\n", fr.Title)
	fmt.Fprintf(&b, "id: %s\n", fr.ID)
	fmt.Fprintf(&b, "date: %s\n", fr.LocalDate)
	fmt.Fprintf(&b, "start: %s\n", fr.StartUTC)
	fmt.Fprintf(&b, "end: %s\n", fr.EndUTC)
	fmt.Fprintf(&b, "duration_min: %.0f\n", fr.DurationMin)
	fmt.Fprintf(&b, "speakers: %s\n", strings.Join(fr.Speakers, ", "))
	fmt.Fprintf(&b, "category: %s\n", fr.Category)
	fmt.Fprintf(&b, "starred: %v\n", fr.IsStarred)
	b.WriteString("---\n\n")
	b.WriteString(fr.TranscriptMD)
	b.WriteString("\n")
	return b.String()
}
