package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
