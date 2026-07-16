package limitless

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// pageBody builds a one-page response with n logs and the given cursor.
func pageBody(t *testing.T, ids []string, next string) string {
	t.Helper()
	logs := make([]Lifelog, len(ids))
	for i, id := range ids {
		logs[i] = Lifelog{ID: id, Title: "t-" + id,
			StartTime: "2026-07-06T14:00:00Z", EndTime: "2026-07-06T14:30:00Z"}
	}
	var resp listResponse
	resp.Data.Lifelogs = logs
	resp.Meta.Lifelogs.NextCursor = next
	resp.Meta.Lifelogs.Count = len(logs)
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func TestFetchAllPaginatesAndSendsParams(t *testing.T) {
	var gotQueries []map[string]string
	var gotKeys []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/lifelogs" {
			t.Errorf("path = %q, want /v1/lifelogs", r.URL.Path)
		}
		gotKeys = append(gotKeys, r.Header.Get("X-API-Key"))
		q := map[string]string{}
		for k := range r.URL.Query() {
			q[k] = r.URL.Query().Get(k)
		}
		gotQueries = append(gotQueries, q)

		switch r.URL.Query().Get("cursor") {
		case "":
			fmt.Fprint(w, pageBody(t, []string{"a1", "a2"}, "cur_2"))
		case "cur_2":
			fmt.Fprint(w, pageBody(t, []string{"a3"}, ""))
		default:
			t.Errorf("unexpected cursor %q", r.URL.Query().Get("cursor"))
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	logs, err := c.FetchAll(context.Background(),
		ListParams{Date: "2026-07-06", Timezone: "America/Chicago", Direction: "asc"})
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(logs) != 3 || logs[0].ID != "a1" || logs[2].ID != "a3" {
		t.Errorf("got %d logs %+v", len(logs), logs)
	}
	if len(gotQueries) != 2 {
		t.Fatalf("made %d requests, want 2", len(gotQueries))
	}
	first := gotQueries[0]
	for k, want := range map[string]string{
		"limit":           "10",
		"includeMarkdown": "true",
		"includeHeadings": "true",
		"includeContents": "true",
		"date":            "2026-07-06",
		"timezone":        "America/Chicago",
		"direction":       "asc",
	} {
		if first[k] != want {
			t.Errorf("param %s = %q, want %q", k, first[k], want)
		}
	}
	if _, hasCursor := first["cursor"]; hasCursor {
		t.Error("first request must not send cursor")
	}
	if gotQueries[1]["cursor"] != "cur_2" {
		t.Errorf("second request cursor = %q, want cur_2", gotQueries[1]["cursor"])
	}
	for _, k := range gotKeys {
		if k != "test-key" {
			t.Errorf("X-API-Key = %q", k)
		}
	}
}

func TestFetchAllAPIErrorIsFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"bad key"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "bad")
	c.Sleep = func(time.Duration) {} // never expected, but keep tests fast
	if _, err := c.FetchAll(context.Background(), ListParams{Date: "2026-07-06"}); err == nil {
		t.Fatal("want error on 401, got nil")
	}
}
