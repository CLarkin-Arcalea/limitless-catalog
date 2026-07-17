package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/catalog"
	"github.com/CLarkin-Arcalea/limitless-catalog/internal/limitless"
	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

type ingestStats struct {
	Inserted, Updated, Skipped int
}

func (a *ingestStats) add(b ingestStats) {
	a.Inserted += b.Inserted
	a.Updated += b.Updated
	a.Skipped += b.Skipped
}

// cmdIngest handles: ingest backfill [--days N | --months N | --all | --from-start [--months N] | --start D --end D]
//
//	ingest incremental
//	ingest --date YYYY-MM-DD
func cmdIngest(cfg config, args []string) error {
	if cfg.apiKey == "" {
		return fmt.Errorf("no API key: pass --api-key or set LIMITLESS_API_KEY")
	}
	client := limitless.NewClient(cfg.baseURL, cfg.apiKey)

	st, err := store.Open(cfg.dbPath)
	if err != nil {
		return err
	}
	defer st.Close()

	ctx := context.Background()
	today := time.Now().In(cfg.loc).Format("2006-01-02")

	mode := ""
	if len(args) > 0 && (args[0] == "backfill" || args[0] == "incremental") {
		mode, args = args[0], args[1:]
	}

	switch mode {
	case "backfill":
		fs := flag.NewFlagSet("ingest backfill", flag.ExitOnError)
		days := fs.Int("days", 90, "how many days back to fetch")
		months := fs.Int("months", 0, "how many months back to fetch (overrides --days)")
		all := fs.Bool("all", false, "everything from the oldest available lifelog")
		fromStart := fs.Bool("from-start", false, "anchor the window at the oldest available lifelog, going forward")
		start := fs.String("start", "", "start date YYYY-MM-DD (overrides other window flags)")
		end := fs.String("end", "", "end date YYYY-MM-DD (requires --start; default today)")
		fs.Parse(args)

		startDate, endDate, err := resolveBackfillWindow(
			*all, *fromStart, *months, *days, *start, *end, today, cfg.loc,
			func() (string, error) { return oldestLocalDate(ctx, client, cfg.loc) })
		if err != nil {
			return err
		}
		return ingestRange(ctx, client, st, cfg, startDate, endDate, today, true)

	case "incremental":
		last, err := st.MaxLocalDate()
		if err != nil {
			return err
		}
		if last == "" {
			return fmt.Errorf("catalog is empty; run: limitless-catalog ingest backfill --days 90")
		}
		// Re-ingest the last stored day too: it may have been partial.
		return ingestRange(ctx, client, st, cfg, last, today, today, false)

	default:
		fs := flag.NewFlagSet("ingest", flag.ExitOnError)
		date := fs.String("date", "", "single date YYYY-MM-DD")
		id := fs.String("id", "", "single lifelog id to fetch and ingest")
		fs.Parse(args)
		if *id != "" {
			l, err := client.GetLifelog(ctx, *id)
			if err != nil {
				return err
			}
			rec, err := catalog.Build(*l, cfg.loc)
			if err != nil {
				return err
			}
			res, err := st.Upsert(rec)
			if err != nil {
				return err
			}
			if err := st.SetState("last_ingest_run", time.Now().UTC().Format(time.RFC3339)); err != nil {
				return err
			}
			fmt.Printf("%s: %s (%s, %s)\n", *id, res, rec.LocalDate, rec.Title)
			return nil
		}
		if *date == "" {
			return fmt.Errorf("usage: ingest backfill [--days N | --months N | --all | --from-start [--months N] | --start D --end D] | ingest incremental | ingest orphans | ingest --date D | ingest --id ID")
		}
		stats, err := ingestDate(ctx, client, st, cfg, *date)
		if err != nil {
			return err
		}
		if err := st.SetState("last_ingest_run", time.Now().UTC().Format(time.RFC3339)); err != nil {
			return err
		}
		fmt.Printf("%s: %d inserted, %d updated, %d skipped\n",
			*date, stats.Inserted, stats.Updated, stats.Skipped)
		return nil
	}
}

// ingestRange walks [startDate, endDate] day by day. With markDone, days
// other than today are recorded in ingest_state and skipped on re-runs.
func ingestRange(ctx context.Context, client *limitless.Client, st *store.Store,
	cfg config, startDate, endDate, today string, markDone bool) error {

	start, err := time.ParseInLocation("2006-01-02", startDate, cfg.loc)
	if err != nil {
		return fmt.Errorf("bad start date %q: %w", startDate, err)
	}
	end, err := time.ParseInLocation("2006-01-02", endDate, cfg.loc)
	if err != nil {
		return fmt.Errorf("bad end date %q: %w", endDate, err)
	}
	if end.Before(start) {
		return fmt.Errorf("end %s is before start %s", endDate, startDate)
	}

	var total ingestStats
	daysDone := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		date := d.Format("2006-01-02")
		stateKey := "backfill_done:" + date

		if markDone {
			if v, err := st.GetState(stateKey); err != nil {
				return err
			} else if v == "1" {
				continue
			}
		}

		stats, err := ingestDate(ctx, client, st, cfg, date)
		if err != nil {
			return fmt.Errorf("ingest %s: %w", date, err)
		}
		total.add(stats)
		daysDone++
		fmt.Printf("%s: %d inserted, %d updated, %d skipped\n",
			date, stats.Inserted, stats.Updated, stats.Skipped)

		// Today is still being recorded; never mark it complete.
		if markDone && date != today {
			if err := st.SetState(stateKey, "1"); err != nil {
				return err
			}
		}
	}
	if err := st.SetState("last_ingest_run", time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	fmt.Printf("done: %d days, %d inserted, %d updated, %d skipped\n",
		daysDone, total.Inserted, total.Updated, total.Skipped)
	return nil
}

// ingestDate fetches one local day and upserts every lifelog in it.
func ingestDate(ctx context.Context, client *limitless.Client, st *store.Store,
	cfg config, date string) (ingestStats, error) {

	logs, err := client.FetchAll(ctx, limitless.ListParams{
		Date: date, Timezone: cfg.tzName, Direction: "asc"})
	if err != nil {
		return ingestStats{}, err
	}

	var stats ingestStats
	for _, l := range logs {
		rec, err := catalog.Build(l, cfg.loc)
		if err != nil {
			return stats, err
		}
		res, err := st.Upsert(rec)
		if err != nil {
			return stats, err
		}
		switch res {
		case store.Inserted:
			stats.Inserted++
		case store.Updated:
			stats.Updated++
		case store.Skipped:
			stats.Skipped++
		}
	}
	return stats, nil
}
