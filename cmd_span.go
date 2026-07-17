package main

import (
	"context"
	"fmt"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/catalog"
	"github.com/CLarkin-Arcalea/limitless-catalog/internal/limitless"
)

// resolveBackfillWindow turns the backfill flags into a [start, end] date
// pair. Precedence: explicit --start/--end, then --all / --from-start
// (anchored at the account's oldest lifelog via oldest()), then --months
// back from today, then --days back from today.
func resolveBackfillWindow(all, fromStart bool, months, days int,
	start, end, today string, loc *time.Location,
	oldest func() (string, error)) (string, string, error) {

	if end != "" && start == "" {
		return "", "", fmt.Errorf("--end requires --start")
	}
	if start != "" {
		e := today
		if end != "" {
			e = end
		}
		return start, e, nil
	}
	if all || fromStart {
		o, err := oldest()
		if err != nil {
			return "", "", err
		}
		if o == "" {
			return "", "", fmt.Errorf("account has no lifelogs to backfill")
		}
		if fromStart && months > 0 {
			t, err := time.ParseInLocation("2006-01-02", o, loc)
			if err != nil {
				return "", "", fmt.Errorf("bad oldest date %q: %w", o, err)
			}
			windowEnd := t.AddDate(0, months, 0).Format("2006-01-02")
			if windowEnd > today {
				windowEnd = today
			}
			return o, windowEnd, nil
		}
		return o, today, nil
	}
	t, err := time.ParseInLocation("2006-01-02", today, loc)
	if err != nil {
		return "", "", err
	}
	if months > 0 {
		return t.AddDate(0, -months, 0).Format("2006-01-02"), today, nil
	}
	return t.AddDate(0, 0, -days).Format("2006-01-02"), today, nil
}

// saneOldestEdge returns the account's oldest lifelog with a plausible
// startTime, skipping corrupted epoch-era records. nil when the account
// has no plausible lifelogs.
func saneOldestEdge(ctx context.Context, client *limitless.Client) (*limitless.Lifelog, error) {
	edge, err := client.FetchEdge(ctx, "asc")
	if err != nil || edge == nil {
		return edge, err
	}
	t, err := time.Parse(time.RFC3339, edge.StartTime)
	if err == nil && t.Format("2006-01-02") >= catalog.SaneFloor {
		return edge, nil
	}
	// Corrupted or unparseable timestamp: ask for the first lifelog at or
	// after the floor instead.
	return client.FetchFirst(ctx, limitless.ListParams{Direction: "asc", Start: catalog.SaneFloor})
}

// oldestLocalDate fetches the account's oldest lifelog and buckets its
// start time into a local date.
func oldestLocalDate(ctx context.Context, client *limitless.Client, loc *time.Location) (string, error) {
	edge, err := saneOldestEdge(ctx, client)
	if err != nil {
		return "", err
	}
	if edge == nil {
		return "", nil
	}
	t, err := time.Parse(time.RFC3339, edge.StartTime)
	if err != nil {
		return "", fmt.Errorf("parse oldest startTime %q: %w", edge.StartTime, err)
	}
	return t.In(loc).Format("2006-01-02"), nil
}

// cmdSpan shows what the API has versus what the local catalog holds.
func cmdSpan(cfg config, args []string) error {
	if cfg.apiKey == "" {
		return fmt.Errorf("no API key: pass --api-key or set LIMITLESS_API_KEY")
	}
	client := limitless.NewClient(cfg.baseURL, cfg.apiKey)
	ctx := context.Background()

	oldest, err := saneOldestEdge(ctx, client)
	if err != nil {
		return err
	}
	newest, err := client.FetchEdge(ctx, "desc")
	if err != nil {
		return err
	}
	if oldest == nil || newest == nil {
		fmt.Println("the API reports no lifelogs for this account")
		return nil
	}
	oldestDate, err := apiLocalDate(oldest.StartTime, cfg.loc)
	if err != nil {
		return err
	}
	newestDate, err := apiLocalDate(newest.StartTime, cfg.loc)
	if err != nil {
		return err
	}

	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer s.Close()
	st, err := s.Stats(cfg.dbPath)
	if err != nil {
		return err
	}

	if cfg.asJSON {
		out, err := jsonIndent(map[string]any{
			"api_oldest": oldestDate, "api_newest": newestDate,
			"local_first": st.FirstDate, "local_last": st.LastDate,
			"local_total": st.Total,
		})
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	fmt.Printf("API span:  %s to %s\n", oldestDate, newestDate)
	if st.Total == 0 {
		fmt.Println("Local:     empty")
	} else {
		fmt.Printf("Local:     %s to %s (%d lifelogs)\n", st.FirstDate, st.LastDate, st.Total)
	}
	if st.Total == 0 || st.FirstDate > oldestDate {
		fmt.Println("\nFetch everything:  limitless-catalog ingest backfill --all")
		fmt.Println("Or a window:       limitless-catalog ingest backfill --months 3")
		fmt.Println("From the start:    limitless-catalog ingest backfill --from-start --months 3")
	}
	return nil
}

func apiLocalDate(rfc3339 string, loc *time.Location) (string, error) {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return "", fmt.Errorf("parse startTime %q: %w", rfc3339, err)
	}
	return t.In(loc).Format("2006-01-02"), nil
}
