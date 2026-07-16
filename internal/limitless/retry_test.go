package limitless

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRetryHonorsRetryAfterOn429(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		fmt.Fprint(w, pageBody(t, []string{"x1"}, ""))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	var slept []time.Duration
	c.Sleep = func(d time.Duration) { slept = append(slept, d) }

	logs, err := c.FetchAll(context.Background(), ListParams{Date: "2026-07-06"})
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(logs) != 1 || calls != 2 {
		t.Errorf("logs=%d calls=%d, want 1 and 2", len(logs), calls)
	}
	if len(slept) != 1 || slept[0] != 7*time.Second {
		t.Errorf("slept %v, want [7s] from Retry-After", slept)
	}
}

func TestRetryBacksOffOn5xxThenSucceeds(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		fmt.Fprint(w, pageBody(t, []string{"x1"}, ""))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	var slept []time.Duration
	c.Sleep = func(d time.Duration) { slept = append(slept, d) }

	if _, err := c.FetchAll(context.Background(), ListParams{Date: "2026-07-06"}); err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	want := []time.Duration{time.Second, 2 * time.Second}
	if len(slept) != len(want) || slept[0] != want[0] || slept[1] != want[1] {
		t.Errorf("slept %v, want %v (doubling backoff)", slept, want)
	}
}

func TestRetryGivesUpAfterMaxAttempts(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	c.Sleep = func(time.Duration) {}

	if _, err := c.FetchAll(context.Background(), ListParams{Date: "2026-07-06"}); err == nil {
		t.Fatal("want error after persistent 500s")
	}
	if calls != maxAttempts {
		t.Errorf("made %d calls, want %d", calls, maxAttempts)
	}
}
