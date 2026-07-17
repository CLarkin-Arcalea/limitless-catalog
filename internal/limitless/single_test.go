package limitless

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetLifelog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/lifelogs/abc123" {
			t.Errorf("path = %q", r.URL.Path)
		}
		q := r.URL.Query()
		for _, k := range []string{"includeMarkdown", "includeHeadings", "includeContents"} {
			if q.Get(k) != "true" {
				t.Errorf("param %s = %q, want true", k, q.Get(k))
			}
		}
		resp := map[string]any{"data": map[string]any{"lifelog": Lifelog{
			ID: "abc123", Title: "One record",
			StartTime: "2026-07-01T10:00:00Z", EndTime: "2026-07-01T10:30:00Z",
		}}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	l, err := c.GetLifelog(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("GetLifelog: %v", err)
	}
	if l == nil || l.ID != "abc123" || l.Title != "One record" {
		t.Errorf("got %+v", l)
	}
}

func TestGetLifelogNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	if _, err := c.GetLifelog(context.Background(), "missing"); err == nil {
		t.Fatal("want error on 404")
	}
}

func TestGetLifelogEmptyID(t *testing.T) {
	c := NewClient("http://unused.invalid", "k")
	if _, err := c.GetLifelog(context.Background(), ""); err == nil {
		t.Fatal("want error on empty id")
	}
}
