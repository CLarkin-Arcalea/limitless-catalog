package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// ExportRecords returns full records (transcript + raw json included) for
// local_date in [startDate, endDate], ascending. Empty strings mean
// unbounded on that side.
func (s *Store) ExportRecords(startDate, endDate string) ([]FullRecord, error) {
	query := `
		SELECT ` + rowCols + `, updated_at, transcript_md, raw_json, ingested_at
		FROM lifelogs WHERE 1=1`
	var args []any
	if startDate != "" {
		query += ` AND local_date >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		query += ` AND local_date <= ?`
		args = append(args, endDate)
	}
	query += ` ORDER BY start_utc ASC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FullRecord
	for rows.Next() {
		fr, err := scanFullRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, fr)
	}
	return out, rows.Err()
}

// ExportRecordsMatching returns full records whose title or transcript
// matches the FTS phrase query, optionally bounded by local_date
// (inclusive, empty means unbounded), ascending by start time.
func (s *Store) ExportRecordsMatching(term, startDate, endDate string) ([]FullRecord, error) {
	phrase := `"` + strings.ReplaceAll(term, `"`, `""`) + `"`
	query := `
		SELECT ` + qualifiedRowCols(`l`) + `, l.updated_at, l.transcript_md, l.raw_json, l.ingested_at
		FROM lifelogs_fts f
		JOIN lifelogs l ON l.id = f.id
		WHERE lifelogs_fts MATCH ?`
	args := []any{phrase}
	if startDate != "" {
		query += ` AND l.local_date >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		query += ` AND l.local_date <= ?`
		args = append(args, endDate)
	}
	query += ` ORDER BY l.start_utc ASC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FullRecord
	for rows.Next() {
		fr, err := scanFullRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, fr)
	}
	return out, rows.Err()
}

func scanFullRecord(rows *sql.Rows) (FullRecord, error) {
	var fr FullRecord
	var speakers string
	var starred int
	err := rows.Scan(&fr.ID, &fr.LocalDate, &fr.Title, &fr.StartUTC, &fr.EndUTC,
		&fr.DurationMin, &speakers, &fr.Category, &starred,
		&fr.UpdatedAt, &fr.TranscriptMD, &fr.RawJSON, &fr.IngestedAt)
	if err != nil {
		return FullRecord{}, err
	}
	fr.IsStarred = starred != 0
	if err := json.Unmarshal([]byte(speakers), &fr.Speakers); err != nil {
		return FullRecord{}, fmt.Errorf("decode speakers for %s: %w", fr.ID, err)
	}
	return fr, nil
}
