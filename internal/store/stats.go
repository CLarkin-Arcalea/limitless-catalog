package store

import (
	"os"
	"time"
)

// MonthCount is lifelogs per YYYY-MM month.
type MonthCount struct {
	Month string `json:"month"`
	Count int    `json:"count"`
}

// Stats summarizes catalog coverage so a user can trust (or fix) a backfill.
type Stats struct {
	Total      int            `json:"total"`
	FirstDate  string         `json:"first_date"`
	LastDate   string         `json:"last_date"`
	TotalHours float64        `json:"total_hours"`
	PerMonth   []MonthCount   `json:"per_month"`
	EmptyDays  []string       `json:"empty_days"`
	ByCategory map[string]int `json:"by_category"`
	LastIngest string         `json:"last_ingest"`
	DBBytes    int64          `json:"db_bytes"`
}

// Stats computes catalog statistics. dbPath (optional) sizes the DB file.
func (s *Store) Stats(dbPath string) (Stats, error) {
	st := Stats{ByCategory: map[string]int{}}

	err := s.db.QueryRow(`
		SELECT COUNT(*),
		       COALESCE(MIN(local_date), ''),
		       COALESCE(MAX(local_date), ''),
		       COALESCE(SUM(duration_min), 0) / 60.0
		FROM lifelogs`).Scan(&st.Total, &st.FirstDate, &st.LastDate, &st.TotalHours)
	if err != nil {
		return st, err
	}

	rows, err := s.db.Query(`
		SELECT substr(local_date, 1, 7) AS month, COUNT(*)
		FROM lifelogs GROUP BY month ORDER BY month`)
	if err != nil {
		return st, err
	}
	for rows.Next() {
		var mc MonthCount
		if err := rows.Scan(&mc.Month, &mc.Count); err != nil {
			rows.Close()
			return st, err
		}
		st.PerMonth = append(st.PerMonth, mc)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return st, err
	}

	rows, err = s.db.Query(`SELECT category, COUNT(*) FROM lifelogs GROUP BY category`)
	if err != nil {
		return st, err
	}
	for rows.Next() {
		var cat string
		var n int
		if err := rows.Scan(&cat, &n); err != nil {
			rows.Close()
			return st, err
		}
		st.ByCategory[cat] = n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return st, err
	}

	if st.FirstDate != "" && st.LastDate != "" {
		covered := map[string]bool{}
		rows, err = s.db.Query(`SELECT DISTINCT local_date FROM lifelogs`)
		if err != nil {
			return st, err
		}
		for rows.Next() {
			var d string
			if err := rows.Scan(&d); err != nil {
				rows.Close()
				return st, err
			}
			covered[d] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return st, err
		}

		first, err := time.Parse("2006-01-02", st.FirstDate)
		if err != nil {
			return st, err
		}
		last, err := time.Parse("2006-01-02", st.LastDate)
		if err != nil {
			return st, err
		}
		for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
			if ds := d.Format("2006-01-02"); !covered[ds] {
				st.EmptyDays = append(st.EmptyDays, ds)
			}
		}
	}

	st.LastIngest, err = s.GetState("last_ingest_run")
	if err != nil {
		return st, err
	}
	if dbPath != "" {
		if fi, err := os.Stat(dbPath); err == nil {
			st.DBBytes = fi.Size()
		}
	}
	return st, nil
}
