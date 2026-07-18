package store

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// SpeakerStat summarizes one speaker's presence across the catalog.
type SpeakerStat struct {
	Name           string `json:"name"`
	Count          int    `json:"count"`
	FirstSeen      string `json:"first_seen"`
	LastSeen       string `json:"last_seen"`
	DaysSinceLast  int    `json:"days_since_last"`
	LongestGapDays int    `json:"longest_gap_days"`
}

type speakerAccum struct {
	count int
	dates []string // distinct local_date values, ascending
}

// Speakers aggregates, per distinct name found in the speakers JSON column
// across all rows, total lifelog count, first/last local_date seen, days
// since last seen relative to the most recent lifelog in the whole catalog,
// and the longest gap in days between consecutive appearances. Derived from
// the existing column at query time rather than a persisted table, since
// every row already carries it. Ranked by count descending, then name.
func (s *Store) Speakers() ([]SpeakerStat, error) {
	rows, err := s.db.Query(`SELECT local_date, speakers FROM lifelogs ORDER BY local_date ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agg := map[string]*speakerAccum{}
	var order []string
	maxDate := ""
	for rows.Next() {
		var date, speakersJSON string
		if err := rows.Scan(&date, &speakersJSON); err != nil {
			return nil, err
		}
		if date > maxDate {
			maxDate = date
		}
		var names []string
		if err := json.Unmarshal([]byte(speakersJSON), &names); err != nil {
			return nil, fmt.Errorf("decode speakers: %w", err)
		}
		for _, name := range names {
			a, ok := agg[name]
			if !ok {
				a = &speakerAccum{}
				agg[name] = a
				order = append(order, name)
			}
			a.count++
			if len(a.dates) == 0 || a.dates[len(a.dates)-1] != date {
				a.dates = append(a.dates, date)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]SpeakerStat, 0, len(order))
	for _, name := range order {
		a := agg[name]
		daysSince, err := daysBetween(a.dates[len(a.dates)-1], maxDate)
		if err != nil {
			return nil, err
		}
		out = append(out, SpeakerStat{
			Name:           name,
			Count:          a.count,
			FirstSeen:      a.dates[0],
			LastSeen:       a.dates[len(a.dates)-1],
			DaysSinceLast:  daysSince,
			LongestGapDays: longestGapDays(a.dates),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// Speaker returns the aggregate stat for one speaker name (exact match, as
// stored), or nil if that name never appears in the catalog.
func (s *Store) Speaker(name string) (*SpeakerStat, error) {
	all, err := s.Speakers()
	if err != nil {
		return nil, err
	}
	for _, st := range all {
		if st.Name == name {
			return &st, nil
		}
	}
	return nil, nil
}

// longestGapDays returns the largest gap, in days, between consecutive
// entries of an ascending, distinct list of YYYY-MM-DD dates. Zero for a
// single date.
func longestGapDays(dates []string) int {
	max := 0
	for i := 1; i < len(dates); i++ {
		if gap, err := daysBetween(dates[i-1], dates[i]); err == nil && gap > max {
			max = gap
		}
	}
	return max
}

// daysBetween returns the whole-day difference between two YYYY-MM-DD dates.
func daysBetween(from, to string) (int, error) {
	f, err := time.Parse("2006-01-02", from)
	if err != nil {
		return 0, err
	}
	t, err := time.Parse("2006-01-02", to)
	if err != nil {
		return 0, err
	}
	return int(t.Sub(f).Hours() / 24), nil
}
