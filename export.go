package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

const redactedPlaceholder = "[REDACTED]"

// speakerRedactor replaces one or more speaker names with a placeholder in
// exported output only; it never touches the database. Built once per
// export run and applied to each in-memory FullRecord before rendering.
type speakerRedactor struct {
	lower map[string]bool
	res   []*regexp.Regexp
}

// newSpeakerRedactor compiles a whole-word, case-insensitive matcher for
// each name. Empty names are ignored.
func newSpeakerRedactor(names []string) *speakerRedactor {
	r := &speakerRedactor{lower: map[string]bool{}}
	for _, n := range names {
		if n == "" {
			continue
		}
		r.lower[strings.ToLower(n)] = true
		r.res = append(r.res, regexp.MustCompile(`(?i)\b`+regexp.QuoteMeta(n)+`\b`))
	}
	return r
}

// redact returns a copy of fr with the target speaker(s) replaced in the
// speakers list, the transcript text, and raw_json (so a JSON export's raw
// payload doesn't quietly leak the name the caller asked to redact).
func (r *speakerRedactor) redact(fr store.FullRecord) store.FullRecord {
	if len(r.res) == 0 {
		return fr
	}
	for _, re := range r.res {
		fr.TranscriptMD = re.ReplaceAllString(fr.TranscriptMD, redactedPlaceholder)
		fr.RawJSON = re.ReplaceAllString(fr.RawJSON, redactedPlaceholder)
	}
	speakers := make([]string, len(fr.Speakers))
	for i, sp := range fr.Speakers {
		if r.lower[strings.ToLower(sp)] {
			speakers[i] = redactedPlaceholder
		} else {
			speakers[i] = sp
		}
	}
	fr.Speakers = speakers
	return fr
}

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
