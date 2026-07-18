package main

import (
	"regexp"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

// piiFinding reports that a pattern matched somewhere in a record, never
// the matched text itself: the report must not become a second copy of
// the sensitive data it flags.
type piiFinding struct {
	LifelogID string `json:"lifelog_id"`
	LocalDate string `json:"local_date"`
	Kind      string `json:"kind"`
}

// piiPatterns are simple stdlib-regexp shape heuristics, not validators:
// they flag things that look like SSNs, credit cards, phone numbers, or
// emails. Expect false positives and false negatives. This is best-effort
// detection to point a human at likely spots, not a compliance guarantee.
var piiPatterns = []struct {
	kind string
	re   *regexp.Regexp
}{
	{"ssn", regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)},
	{"credit_card", regexp.MustCompile(`\b(?:\d{4}[- ]?){3}\d{4}\b`)},
	{"email", regexp.MustCompile(`\b[\w.+-]+@[\w-]+\.[\w-]+(?:\.[\w-]+)*\b`)},
	{"phone", regexp.MustCompile(`\b(?:\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]\d{3}[-.\s]\d{4}\b`)},
}

// scanForPII checks a record's title and transcript against piiPatterns,
// returning at most one finding per pattern kind per record.
func scanForPII(fr store.FullRecord) []piiFinding {
	text := fr.Title + "\n" + fr.TranscriptMD
	var out []piiFinding
	for _, p := range piiPatterns {
		if p.re.MatchString(text) {
			out = append(out, piiFinding{LifelogID: fr.ID, LocalDate: fr.LocalDate, Kind: p.kind})
		}
	}
	return out
}
