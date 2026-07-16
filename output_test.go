package main

import (
	"strings"
	"testing"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

func sampleRows() []store.Row {
	return []store.Row{{
		ID: "abc12345", LocalDate: "2026-07-06", Title: "Ben 1:1",
		StartUTC: "2026-07-06T18:00:00Z", EndUTC: "2026-07-06T18:45:00Z",
		DurationMin: 45, Speakers: []string{"Ava", "Ben"},
		Category: "unknown", Snippet: "[kubernetes] migration",
	}}
}

func TestFormatRowsText(t *testing.T) {
	out, err := formatRows(sampleRows(), false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"abc12345", "2026-07-06", "Ben 1:1", "45m",
		"Ava, Ben", "[kubernetes] migration"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q in:\n%s", want, out)
		}
	}
}

func TestFormatRowsJSON(t *testing.T) {
	out, err := formatRows(sampleRows(), true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"id": "abc12345"`) {
		t.Errorf("json output wrong:\n%s", out)
	}
}

func TestFormatRowsEmpty(t *testing.T) {
	out, err := formatRows(nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no results") {
		t.Errorf("empty output = %q", out)
	}
}

func TestCmdRecentRejectsBadArgs(t *testing.T) {
	cfg := config{dbPath: "unused.db"}
	for _, args := range [][]string{
		{"5", "--json"},
		{"-5"},
		{"0"},
		{"abc"},
	} {
		if err := cmdRecent(cfg, args); err == nil {
			t.Errorf("cmdRecent(%v): want usage error, got nil", args)
		}
	}
}

func TestParseLocalDateTime(t *testing.T) {
	chicago, _ := time.LoadLocation("America/Chicago")
	for _, in := range []string{"2026-07-06 13:00", "2026-07-06T13:00"} {
		got, err := parseLocalDateTime(in, chicago)
		if err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}
		if got.UTC().Format(time.RFC3339) != "2026-07-06T18:00:00Z" {
			t.Errorf("%q parsed to %v", in, got.UTC())
		}
	}
	if _, err := parseLocalDateTime("garbage", chicago); err == nil {
		t.Error("want error for garbage input")
	}
}
