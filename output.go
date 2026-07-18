package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

// formatRows renders listing rows as aligned text or indented JSON.
func formatRows(rows []store.Row, asJSON bool) (string, error) {
	if asJSON {
		b, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	if len(rows) == 0 {
		return "no results\n", nil
	}
	var b strings.Builder
	for _, r := range rows {
		star := " "
		if r.IsStarred {
			star = "*"
		}
		fmt.Fprintf(&b, "%s %s  %s  %3.0fm  %-10s %s\n",
			star, r.LocalDate, r.ID, r.DurationMin, r.Category, r.Title)
		if len(r.Speakers) > 0 {
			fmt.Fprintf(&b, "    speakers: %s\n", strings.Join(r.Speakers, ", "))
		}
		if r.Snippet != "" {
			fmt.Fprintf(&b, "    %s\n", r.Snippet)
		}
	}
	return b.String(), nil
}

// formatOnThisDay renders rows (expected newest-year-first from
// Store.OnThisDay) grouped under a year heading, or indented JSON.
func formatOnThisDay(rows []store.Row, loc *time.Location, asJSON bool) (string, error) {
	if asJSON {
		b, err := json.MarshalIndent(rows, "", "  ")
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	if len(rows) == 0 {
		return "no matches\n", nil
	}
	var b strings.Builder
	year := ""
	for _, r := range rows {
		if len(r.LocalDate) >= 4 && r.LocalDate[:4] != year {
			year = r.LocalDate[:4]
			fmt.Fprintf(&b, "%s\n", year)
		}
		clock := r.StartUTC
		if t, err := time.Parse(time.RFC3339, r.StartUTC); err == nil {
			clock = t.In(loc).Format("15:04")
		}
		fmt.Fprintf(&b, "  %s  %s\n", clock, r.Title)
		if len(r.Speakers) > 0 {
			fmt.Fprintf(&b, "      speakers: %s\n", strings.Join(r.Speakers, ", "))
		}
	}
	return b.String(), nil
}

// parseLocalDateTime accepts "YYYY-MM-DD HH:MM", "YYYY-MM-DDTHH:MM", or
// RFC3339, interpreted in loc.
func parseLocalDateTime(s string, loc *time.Location) (time.Time, error) {
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02T15:04", time.RFC3339} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized datetime %q (use \"YYYY-MM-DD HH:MM\")", s)
}

// jsonIndent renders any value as indented JSON.
func jsonIndent(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
