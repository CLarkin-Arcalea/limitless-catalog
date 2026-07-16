package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/limitless"
)

// edgePage builds a one-lifelog API response with the given id and startTime.
func edgePage(id, startTime string) string {
	return fmt.Sprintf(`{"data":{"lifelogs":[{"id":%q,"startTime":%q}]},"meta":{"lifelogs":{"nextCursor":"","count":1}}}`,
		id, startTime)
}

func TestSaneOldestEdgeSkipsEpochGarbage(t *testing.T) {
	t.Run("epoch garbage triggers floored refetch", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("start") == "2013-01-01" {
				fmt.Fprint(w, edgePage("real", "2025-03-10T17:23:46Z"))
				return
			}
			fmt.Fprint(w, edgePage("garbage", "1970-01-01T00:00:00Z"))
		}))
		defer srv.Close()

		edge, err := saneOldestEdge(context.Background(), limitless.NewClient(srv.URL, "k"))
		if err != nil {
			t.Fatal(err)
		}
		if edge == nil || edge.ID != "real" || edge.StartTime != "2025-03-10T17:23:46Z" {
			t.Errorf("got %+v, want the 2025 lifelog from the floored refetch", edge)
		}
	})

	t.Run("sane edge needs no refetch", func(t *testing.T) {
		var requests int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			fmt.Fprint(w, edgePage("sane", "2025-03-10T17:23:46Z"))
		}))
		defer srv.Close()

		edge, err := saneOldestEdge(context.Background(), limitless.NewClient(srv.URL, "k"))
		if err != nil {
			t.Fatal(err)
		}
		if edge == nil || edge.ID != "sane" {
			t.Errorf("got %+v, want the sane edge", edge)
		}
		if requests != 1 {
			t.Errorf("made %d requests, want 1 (no refetch for a sane edge)", requests)
		}
	})

	t.Run("garbage on both calls returns floored result without looping", func(t *testing.T) {
		var requests int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			fmt.Fprint(w, edgePage("garbage", "1970-01-01T00:00:00Z"))
		}))
		defer srv.Close()

		edge, err := saneOldestEdge(context.Background(), limitless.NewClient(srv.URL, "k"))
		if err != nil {
			t.Fatal(err)
		}
		if edge == nil || edge.ID != "garbage" {
			t.Errorf("got %+v, want the floored call's result surfaced as-is", edge)
		}
		if requests != 2 {
			t.Errorf("made %d requests, want exactly 2 (one refetch, no loop)", requests)
		}
	})
}

func TestResolveBackfillWindow(t *testing.T) {
	loc := time.UTC
	today := "2026-07-07"
	oldestOK := func() (string, error) { return "2025-11-14", nil }

	cases := []struct {
		name               string
		all, fromStart     bool
		months, days       int
		start, end         string
		wantStart, wantEnd string
	}{
		{"months back from now", false, false, 3, 90, "", "", "2026-04-07", "2026-07-07"},
		{"days when no months", false, false, 0, 30, "", "", "2026-06-07", "2026-07-07"},
		{"explicit start/end wins", false, false, 0, 90, "2026-01-01", "2026-02-01", "2026-01-01", "2026-02-01"},
		{"all from oldest", true, false, 0, 90, "", "", "2025-11-14", "2026-07-07"},
		{"from start with months", false, true, 3, 90, "", "", "2025-11-14", "2026-02-14"},
		{"from start no months means all", false, true, 0, 90, "", "", "2025-11-14", "2026-07-07"},
	}
	for _, c := range cases {
		gotStart, gotEnd, err := resolveBackfillWindow(
			c.all, c.fromStart, c.months, c.days, c.start, c.end, today, loc, oldestOK)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if gotStart != c.wantStart || gotEnd != c.wantEnd {
			t.Errorf("%s: got %s..%s, want %s..%s",
				c.name, gotStart, gotEnd, c.wantStart, c.wantEnd)
		}
	}
}

func TestResolveBackfillWindowLoneEndErrors(t *testing.T) {
	oldest := func() (string, error) { return "2025-11-14", nil }
	if _, _, err := resolveBackfillWindow(false, false, 0, 90, "", "2026-06-30", "2026-07-07", time.UTC, oldest); err == nil {
		t.Error("lone --end must error, not be silently ignored")
	}
}

func TestResolveBackfillWindowEmptyAccount(t *testing.T) {
	noLogs := func() (string, error) { return "", nil }
	_, _, err := resolveBackfillWindow(true, false, 0, 90, "", "", "2026-07-07", time.UTC, noLogs)
	if err == nil {
		t.Error("want error when account has no lifelogs")
	}
}
