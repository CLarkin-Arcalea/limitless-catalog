package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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
		if strings.HasPrefix(r.URL.Path, "/v1/lifelogs/") && r.URL.Path != "/v1/lifelogs/" {
			id := strings.TrimPrefix(r.URL.Path, "/v1/lifelogs/")
			*requestLog = append(*requestLog, "byid|"+id)
			body, _ := json.Marshal(map[string]any{"data": map[string]any{"lifelog": map[string]any{
				"id": id, "title": "t-" + id,
				"markdown":  "# t-" + id + "\n\n**Ava:** content " + id,
				"startTime": "2026-07-05T14:00:00Z", "endTime": "2026-07-05T14:30:00Z",
				"updatedAt": "2026-07-05T15:00:00Z",
			}}})
			fmt.Fprint(w, string(body))
			return
		}
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

func TestIngestByID(t *testing.T) {
	var reqs []string
	srv := fakeAPI(t, &reqs)
	defer srv.Close()
	cfg := testCfg(t, srv.URL)

	if err := cmdIngest(cfg, []string{"--id", "solo99"}); err != nil {
		t.Fatalf("ingest --id: %v", err)
	}
	s, err := store.Open(cfg.dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	fr, err := s.Get("solo99")
	if err != nil || fr == nil {
		t.Fatalf("record not ingested: %v %v", fr, err)
	}
	if v, _ := s.GetState("last_ingest_run"); v == "" {
		t.Error("--id mode must set last_ingest_run")
	}
}

func TestIngestOrphans(t *testing.T) {
	page := func(logs []map[string]any, next string) string {
		body, _ := json.Marshal(map[string]any{
			"data": map[string]any{"lifelogs": logs},
			"meta": map[string]any{"lifelogs": map[string]any{"nextCursor": next, "count": len(logs)}},
		})
		return string(body)
	}
	orphan := func(id, start string) map[string]any {
		return map[string]any{
			"id": id, "title": "t-" + id,
			"markdown":  "# t-" + id + "\n\n**Ava:** orphan content",
			"startTime": start, "endTime": "2025-09-25T21:14:59Z",
			"updatedAt": "2025-09-25T22:00:00Z",
		}
	}
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Query().Get("cursor") {
		case "":
			fmt.Fprint(w, page([]map[string]any{
				orphan("orph1", "1970-01-01T00:00:00Z"),
				orphan("orph2", "1970-01-01T00:00:07Z"),
			}, "c2"))
		case "c2":
			fmt.Fprint(w, page([]map[string]any{
				orphan("sane1", "2025-03-10T17:23:46Z"), // sane: sweep must stop here
			}, "c3"))
		default:
			t.Errorf("sweep did not stop at first sane record (cursor %q)", r.URL.Query().Get("cursor"))
			fmt.Fprint(w, page(nil, ""))
		}
	}))
	defer srv.Close()
	cfg := testCfg(t, srv.URL)

	if err := cmdIngest(cfg, []string{"orphans"}); err != nil {
		t.Fatalf("orphans: %v", err)
	}
	s, err := store.Open(cfg.dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, id := range []string{"orph1", "orph2"} {
		fr, err := s.Get(id)
		if err != nil || fr == nil {
			t.Errorf("orphan %s not ingested", id)
			continue
		}
		if fr.LocalDate != "2025-09-25" {
			t.Errorf("orphan %s local_date = %q, want repaired 2025-09-25", id, fr.LocalDate)
		}
	}
	if fr, _ := s.Get("sane1"); fr != nil {
		t.Error("sane record must NOT be ingested by the orphan sweep")
	}
	if calls != 2 {
		t.Errorf("made %d page calls, want 2 (stop at first sane)", calls)
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
