// Package catalog converts raw API lifelogs into normalized records
// ready for storage: assembled transcript, speakers, timestamps, category.
package catalog

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/limitless"
)

// Record is one lifelog normalized for the store.
type Record struct {
	ID           string
	StartUTC     string // RFC3339 UTC, Z suffix
	EndUTC       string
	LocalDate    string // YYYY-MM-DD in the configured timezone
	Title        string
	DurationMin  float64
	IsStarred    bool
	UpdatedAt    string // RFC3339 UTC when parseable, else raw API value
	Speakers     []string
	TranscriptMD string
	Category     string // work | personal | media | unknown
	RawJSON      string
}

// NormalizeUTC parses an ISO-8601/RFC3339 timestamp and returns it as an
// RFC3339 UTC string (Z suffix, second precision) plus the parsed time.
// Stored strings compare correctly with plain lexicographic SQL.
func NormalizeUTC(s string) (string, time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("parse timestamp %q: %w", s, err)
	}
	u := t.UTC()
	return u.Format(time.RFC3339), u, nil
}

// LocalDate buckets t into a YYYY-MM-DD day in loc.
func LocalDate(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("2006-01-02")
}

// DurationMinutes is the span between start and end in minutes.
func DurationMinutes(start, end time.Time) float64 {
	return end.Sub(start).Minutes()
}

// Build converts an API lifelog into a normalized Record, bucketing
// local_date in loc.
func Build(l limitless.Lifelog, loc *time.Location) (Record, error) {
	startStr, startT, err := NormalizeUTC(l.StartTime)
	if err != nil || startStr < SaneFloor {
		// Corrupted or missing start: repair from content nodes / end.
		repaired, ok := repairStart(l)
		if !ok {
			return Record{}, fmt.Errorf("lifelog %s: no usable timestamp (start %q)", l.ID, l.StartTime)
		}
		startT = repaired.UTC()
		startStr = startT.Format(time.RFC3339)
	}
	endStr, endT, err := NormalizeUTC(l.EndTime)
	if err != nil || endStr < SaneFloor {
		// End unusable/corrupted but start recovered: store a point-in-time record.
		endStr, endT = startStr, startT
	}
	duration := DurationMinutes(startT, endT)
	if duration < 0 {
		duration = 0
	}
	updated := l.UpdatedAt
	if norm, _, err := NormalizeUTC(l.UpdatedAt); err == nil {
		updated = norm
	}

	transcript := AssembleTranscript(l)
	speakers := ExtractSpeakers(l)

	raw, err := json.Marshal(l)
	if err != nil {
		return Record{}, fmt.Errorf("lifelog %s marshal raw: %w", l.ID, err)
	}

	return Record{
		ID:           l.ID,
		StartUTC:     startStr,
		EndUTC:       endStr,
		LocalDate:    LocalDate(startT, loc),
		Title:        l.Title,
		DurationMin:  duration,
		IsStarred:    l.IsStarred,
		UpdatedAt:    updated,
		Speakers:     speakers,
		TranscriptMD: transcript,
		Category:     Categorize(transcript, HasUserSpeaker(l)),
		RawJSON:      string(raw),
	}, nil
}
