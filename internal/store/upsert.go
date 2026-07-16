package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/catalog"
)

// UpsertResult says what Upsert did with a record.
type UpsertResult int

const (
	Inserted UpsertResult = iota
	Updated
	Skipped
)

func (r UpsertResult) String() string {
	switch r {
	case Inserted:
		return "inserted"
	case Updated:
		return "updated"
	case Skipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// Upsert writes r keyed by id. Existing rows with an identical non-empty
// updated_at are skipped; changed rows are rewritten along with their FTS
// entry, so re-ingesting is always safe and cheap.
func (s *Store) Upsert(r catalog.Record) (UpsertResult, error) {
	speakers, err := json.Marshal(r.Speakers)
	if err != nil {
		return Skipped, fmt.Errorf("marshal speakers: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return Skipped, err
	}
	defer tx.Rollback()

	var existing string
	err = tx.QueryRow(`SELECT updated_at FROM lifelogs WHERE id = ?`, r.ID).Scan(&existing)
	exists := true
	if errors.Is(err, sql.ErrNoRows) {
		exists = false
	} else if err != nil {
		return Skipped, err
	}
	if exists && r.UpdatedAt != "" && existing == r.UpdatedAt {
		return Skipped, nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.Exec(`
		INSERT INTO lifelogs
			(id, start_utc, end_utc, local_date, title, duration_min, is_starred,
			 updated_at, speakers, transcript_md, category, ingested_at, raw_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			start_utc=excluded.start_utc, end_utc=excluded.end_utc,
			local_date=excluded.local_date, title=excluded.title,
			duration_min=excluded.duration_min, is_starred=excluded.is_starred,
			updated_at=excluded.updated_at, speakers=excluded.speakers,
			transcript_md=excluded.transcript_md, category=excluded.category,
			ingested_at=excluded.ingested_at, raw_json=excluded.raw_json`,
		r.ID, r.StartUTC, r.EndUTC, r.LocalDate, r.Title, r.DurationMin,
		boolToInt(r.IsStarred), r.UpdatedAt, string(speakers), r.TranscriptMD,
		r.Category, now, r.RawJSON); err != nil {
		return Skipped, fmt.Errorf("upsert lifelog %s: %w", r.ID, err)
	}

	if _, err := tx.Exec(`DELETE FROM lifelogs_fts WHERE id = ?`, r.ID); err != nil {
		return Skipped, fmt.Errorf("fts delete %s: %w", r.ID, err)
	}
	if _, err := tx.Exec(
		`INSERT INTO lifelogs_fts (title, transcript_md, id) VALUES (?, ?, ?)`,
		r.Title, r.TranscriptMD, r.ID); err != nil {
		return Skipped, fmt.Errorf("fts insert %s: %w", r.ID, err)
	}

	if err := tx.Commit(); err != nil {
		return Skipped, err
	}
	if exists {
		return Updated, nil
	}
	return Inserted, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
