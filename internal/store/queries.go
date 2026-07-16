package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Row is the concise metadata shape every listing query returns.
// Full transcripts only come from Get, on purpose: no accidental dumps.
type Row struct {
	ID          string   `json:"id"`
	LocalDate   string   `json:"local_date"`
	Title       string   `json:"title"`
	StartUTC    string   `json:"start_utc"`
	EndUTC      string   `json:"end_utc"`
	DurationMin float64  `json:"duration_min"`
	Speakers    []string `json:"speakers"`
	Category    string   `json:"category"`
	IsStarred   bool     `json:"is_starred"`
	Snippet     string   `json:"snippet,omitempty"`
}

// FullRecord is one lifelog with its transcript, from Get/export.
type FullRecord struct {
	Row
	UpdatedAt    string `json:"updated_at"`
	TranscriptMD string `json:"transcript_md"`
	RawJSON      string `json:"raw_json,omitempty"`
	IngestedAt   string `json:"ingested_at"`
}

const rowCols = `id, local_date, title, start_utc, end_utc,
	duration_min, speakers, category, is_starred`

func scanRow(scanner interface{ Scan(...any) error }, withSnippet bool) (Row, error) {
	var r Row
	var speakers string
	var starred int
	dest := []any{&r.ID, &r.LocalDate, &r.Title, &r.StartUTC, &r.EndUTC,
		&r.DurationMin, &speakers, &r.Category, &starred}
	if withSnippet {
		dest = append(dest, &r.Snippet)
	}
	if err := scanner.Scan(dest...); err != nil {
		return Row{}, err
	}
	r.IsStarred = starred != 0
	if err := json.Unmarshal([]byte(speakers), &r.Speakers); err != nil {
		return Row{}, fmt.Errorf("decode speakers for %s: %w", r.ID, err)
	}
	return r, nil
}

func collectRows(rows *sql.Rows, withSnippet bool) ([]Row, error) {
	defer rows.Close()
	var out []Row
	for rows.Next() {
		r, err := scanRow(rows, withSnippet)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Search runs a full-text phrase search over titles and transcripts,
// newest first. The term is phrase-quoted so FTS syntax characters in
// user input cannot break the query.
func (s *Store) Search(term string, limit int) ([]Row, error) {
	phrase := `"` + strings.ReplaceAll(term, `"`, `""`) + `"`
	rows, err := s.db.Query(`
		SELECT l.id, l.local_date, l.title, l.start_utc, l.end_utc,
		       l.duration_min, l.speakers, l.category, l.is_starred,
		       snippet(lifelogs_fts, 1, '[', ']', '…', 12)
		FROM lifelogs_fts f
		JOIN lifelogs l ON l.id = f.id
		WHERE lifelogs_fts MATCH ?
		ORDER BY l.start_utc DESC
		LIMIT ?`, phrase, limit)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, true)
}

// ByDate lists logs whose local_date equals date, in start order.
func (s *Store) ByDate(date string) ([]Row, error) {
	return s.ByRange(date, date)
}

// ByRange lists logs with local_date in [startDate, endDate], in start order.
func (s *Store) ByRange(startDate, endDate string) ([]Row, error) {
	rows, err := s.db.Query(`
		SELECT `+rowCols+` FROM lifelogs
		WHERE local_date >= ? AND local_date <= ?
		ORDER BY start_utc ASC`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, false)
}

// Recent lists the n newest logs.
func (s *Store) Recent(n int) ([]Row, error) {
	rows, err := s.db.Query(`
		SELECT `+rowCols+` FROM lifelogs
		ORDER BY start_utc DESC LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, false)
}

// Meeting lists logs overlapping [start-buffer, end+buffer]. Overlap, not
// containment: segments that begin before or run past the window count.
func (s *Store) Meeting(start, end time.Time, buffer time.Duration) ([]Row, error) {
	ws := start.Add(-buffer).UTC().Format(time.RFC3339)
	we := end.Add(buffer).UTC().Format(time.RFC3339)
	rows, err := s.db.Query(`
		SELECT `+rowCols+` FROM lifelogs
		WHERE start_utc <= ? AND end_utc >= ?
		ORDER BY start_utc ASC`, we, ws)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, false)
}

// Get returns one full record, or nil when the id is unknown.
func (s *Store) Get(id string) (*FullRecord, error) {
	row := s.db.QueryRow(`
		SELECT `+rowCols+`, updated_at, transcript_md, raw_json, ingested_at
		FROM lifelogs WHERE id = ?`, id)

	var fr FullRecord
	var speakers string
	var starred int
	err := row.Scan(&fr.ID, &fr.LocalDate, &fr.Title, &fr.StartUTC, &fr.EndUTC,
		&fr.DurationMin, &speakers, &fr.Category, &starred,
		&fr.UpdatedAt, &fr.TranscriptMD, &fr.RawJSON, &fr.IngestedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fr.IsStarred = starred != 0
	if err := json.Unmarshal([]byte(speakers), &fr.Speakers); err != nil {
		return nil, fmt.Errorf("decode speakers for %s: %w", fr.ID, err)
	}
	return &fr, nil
}

// GetState reads an ingest_state value; empty string when absent.
func (s *Store) GetState(key string) (string, error) {
	var v string
	err := s.db.QueryRow(`SELECT value FROM ingest_state WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

// SetState writes an ingest_state value.
func (s *Store) SetState(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO ingest_state (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// MaxLocalDate returns the newest local_date in the catalog, empty if none.
func (s *Store) MaxLocalDate() (string, error) {
	var d sql.NullString
	if err := s.db.QueryRow(`SELECT MAX(local_date) FROM lifelogs`).Scan(&d); err != nil {
		return "", err
	}
	return d.String, nil
}
