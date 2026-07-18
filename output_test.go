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

func onThisDaySampleRows() []store.Row {
	return []store.Row{
		{ID: "y25", LocalDate: "2025-07-15", Title: "Last year's call",
			StartUTC: "2025-07-15T16:00:00Z", EndUTC: "2025-07-15T16:20:00Z",
			DurationMin: 20, Speakers: []string{"Ben"}, Category: "unknown"},
		{ID: "x24", LocalDate: "2024-07-15", Title: "Old birthday call",
			StartUTC: "2024-07-15T14:00:00Z", EndUTC: "2024-07-15T14:30:00Z",
			DurationMin: 30, Speakers: []string{"Ava"}, Category: "unknown"},
	}
}

func TestFormatOnThisDayText(t *testing.T) {
	out, err := formatOnThisDay(onThisDaySampleRows(), time.UTC, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"2025", "16:00", "Last year's call", "Ben",
		"2024", "14:00", "Old birthday call", "Ava"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q in:\n%s", want, out)
		}
	}
	if strings.Index(out, "2025") > strings.Index(out, "2024") {
		t.Errorf("expected 2025 group before 2024 group:\n%s", out)
	}
}

func TestFormatOnThisDayJSON(t *testing.T) {
	out, err := formatOnThisDay(onThisDaySampleRows(), time.UTC, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"id": "y25"`) {
		t.Errorf("json output wrong:\n%s", out)
	}
}

func TestFormatOnThisDayEmpty(t *testing.T) {
	out, err := formatOnThisDay(nil, time.UTC, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no matches") {
		t.Errorf("empty output = %q", out)
	}
}

func TestCmdOnThisDayRejectsBadArgs(t *testing.T) {
	cfg := config{dbPath: "unused.db", loc: time.UTC}
	for _, args := range [][]string{
		{"extra-arg"},
		{"--date", "2026-07-15", "extra"},
	} {
		if err := cmdOnThisDay(cfg, args); err == nil {
			t.Errorf("cmdOnThisDay(%v): want usage error, got nil", args)
		}
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
