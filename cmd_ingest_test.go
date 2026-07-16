package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

// fakeAPI serves two lifelogs for 2026-07-05 (paginated) and one for 2026-07-06.
func fakeAPI(t *testing.T, requestLog *[]string) *httptest.Server {
	t.Helper()
	page := func(date string, ids []string, next string) string {
		type lifelog struct {
			ID, Title, Markdown, StartTime, EndTime, UpdatedAt string
		}
		logs := make([]map[string]any, len(ids))
		for i, id := range ids {
			logs[i] = map[string]any{
				"id": id, "title": "t-" + id,
				"markdown":  "# t-" + id + "\n\n**Ava:** content " + id,
				"startTime": date + "T14:0" + fmt.Sprint(i) + ":00Z",
				"endTime":   date + "T14:3" + fmt.Sprint(i) + ":00Z",
				"updatedAt": date + "T15:00:00Z",
			}
		}
		body, _ := json.Marshal(map[string]any{
			"data": map[string]any{"lifelogs": logs},
			"meta": map[string]any{"lifelogs": map[string]any{"nextCursor": next, "count": len(logs)}},
		})
		_ = lifelog{}
		return string(body)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		*requestLog = append(*requestLog, q.Get("date")+"|"+q.Get("cursor"))
		switch {
		case q.Get("date") == "2026-07-05" && q.Get("cursor") == "":
			fmt.Fprint(w, page("2026-07-05", []string{"d5a"}, "c2"))
		case q.Get("date") == "2026-07-05" && q.Get("cursor") == "c2":
			fmt.Fprint(w, page("2026-07-05", []string{"d5b"}, ""))
		case q.Get("date") == "2026-07-06":
			body := page("2026-07-06", []string{"d6a"}, "")
			fmt.Fprint(w, body)
		default:
			fmt.Fprint(w, page(q.Get("date"), nil, ""))
		}
	}))
}

func testCfg(t *testing.T, baseURL string) config {
	t.Helper()
	cfg, err := resolveConfig(
		filepath.Join(t.TempDir(), "test.db"), "test-key", baseURL, "UTC", false)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestIngestBackfillRangeAndResume(t *testing.T) {
	var reqs []string
	srv := fakeAPI(t, &reqs)
	defer srv.Close()
	cfg := testCfg(t, srv.URL)

	err := cmdIngest(cfg, []string{"backfill", "--start", "2026-07-05", "--end", "2026-07-06"})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}

	s, err := store.Open(cfg.dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	rows, err := s.ByRange("2026-07-05", "2026-07-06")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("ingested %d rows, want 3: %+v", len(rows), rows)
	}
	if v, _ := s.GetState("backfill_done:2026-07-05"); v != "1" {
		t.Error("2026-07-05 not marked done")
	}
	s.Close()

	// Re-run: completed dates are skipped, so no new API calls for them.
	before := len(reqs)
	if err := cmdIngest(cfg, []string{"backfill", "--start", "2026-07-05", "--end", "2026-07-06"}); err != nil {
		t.Fatalf("re-run: %v", err)
	}
	for _, r := range reqs[before:] {
		if r[:10] == "2026-07-05" || r[:10] == "2026-07-06" {
			t.Errorf("re-run refetched completed date: %s", r)
		}
	}
}

func TestIngestSingleDate(t *testing.T) {
	var reqs []string
	srv := fakeAPI(t, &reqs)
	defer srv.Close()
	cfg := testCfg(t, srv.URL)

	if err := cmdIngest(cfg, []string{"--date", "2026-07-06"}); err != nil {
		t.Fatalf("single date: %v", err)
	}
	s, _ := store.Open(cfg.dbPath)
	defer s.Close()
	rows, _ := s.ByDate("2026-07-06")
	if len(rows) != 1 || rows[0].ID != "d6a" {
		t.Errorf("got %+v", rows)
	}
	if v, _ := s.GetState("last_ingest_run"); v == "" {
		t.Error("single-date ingest must set last_ingest_run")
	}
}

func TestIngestIncrementalNeedsData(t *testing.T) {
	var reqs []string
	srv := fakeAPI(t, &reqs)
	defer srv.Close()
	cfg := testCfg(t, srv.URL)

	if err := cmdIngest(cfg, []string{"incremental"}); err == nil {
		t.Error("incremental on empty DB should error with backfill suggestion")
	}
}

func TestIngestRequiresAPIKey(t *testing.T) {
	cfg := testCfg(t, "http://unused")
	cfg.apiKey = ""
	if err := cmdIngest(cfg, []string{"--date", "2026-07-06"}); err == nil {
		t.Error("want error when API key missing")
	}
}
