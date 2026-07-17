package limitless

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchEdge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("direction") {
		case "asc":
			fmt.Fprint(w, pageBody(t, []string{"oldest"}, "ignored-cursor"))
		case "desc":
			fmt.Fprint(w, pageBody(t, []string{"newest"}, ""))
		default:
			t.Errorf("missing direction param")
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	oldest, err := c.FetchEdge(context.Background(), "asc")
	if err != nil || oldest == nil || oldest.ID != "oldest" {
		t.Errorf("asc edge: %+v err=%v", oldest, err)
	}
	newest, err := c.FetchEdge(context.Background(), "desc")
	if err != nil || newest == nil || newest.ID != "newest" {
		t.Errorf("desc edge: %+v err=%v", newest, err)
	}
}

func TestFetchFirstSendsStart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("start"); got != "2013-01-01" {
			t.Errorf("start param = %q, want 2013-01-01", got)
		}
		if got := r.URL.Query().Get("direction"); got != "asc" {
			t.Errorf("direction param = %q, want asc", got)
		}
		fmt.Fprint(w, pageBody(t, []string{"first", "second"}, "ignored-cursor"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	got, err := c.FetchFirst(context.Background(), ListParams{Direction: "asc", Start: "2013-01-01"})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != "first" {
		t.Errorf("FetchFirst = %+v, want first item", got)
	}
}

func TestFetchPageExported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cursor") != "c9" {
			t.Errorf("cursor = %q, want c9", r.URL.Query().Get("cursor"))
		}
		fmt.Fprint(w, pageBody(t, []string{"p1", "p2"}, "c10"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	logs, next, err := c.FetchPage(context.Background(), ListParams{Direction: "asc"}, "c9")
	if err != nil || len(logs) != 2 || next != "c10" {
		t.Errorf("logs=%d next=%q err=%v", len(logs), next, err)
	}
}

func TestFetchEdgeEmptyAccount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, pageBody(t, nil, ""))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "k")
	edge, err := c.FetchEdge(context.Background(), "asc")
	if err != nil {
		t.Fatal(err)
	}
	if edge != nil {
		t.Errorf("want nil for empty account, got %+v", edge)
	}
}
