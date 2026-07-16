package main

import (
	"testing"
	"time"
)

func TestResolveTZCoherence(t *testing.T) {
	chicago, _ := time.LoadLocation("America/Chicago")

	loc, name, err := resolveTZ("", chicago)
	if err != nil || loc != chicago || name != "America/Chicago" {
		t.Errorf("real local zone: loc=%v name=%q err=%v", loc, name, err)
	}

	unnamed := time.FixedZone("Local", -5*3600)
	loc, name, err = resolveTZ("", unnamed)
	if err != nil || loc != time.UTC || name != "UTC" {
		t.Errorf("unknown local zone must fall back to UTC for BOTH: loc=%v name=%q err=%v", loc, name, err)
	}

	loc, name, err = resolveTZ("America/Chicago", time.UTC)
	if err != nil || name != "America/Chicago" || loc.String() != "America/Chicago" {
		t.Errorf("explicit flag: loc=%v name=%q err=%v", loc, name, err)
	}

	if _, _, err := resolveTZ("Not/AZone", time.UTC); err == nil {
		t.Error("bad flag must error")
	}
}

func TestResolveConfigAPIKeyPrecedence(t *testing.T) {
	t.Setenv("LIMITLESS_API_KEY", "from-env")

	cfg, err := resolveConfig("db.db", "", "", "UTC", false)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.apiKey != "from-env" {
		t.Errorf("apiKey = %q, want from-env", cfg.apiKey)
	}

	cfg, err = resolveConfig("db.db", "from-flag", "", "UTC", false)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.apiKey != "from-flag" {
		t.Errorf("flag must win, got %q", cfg.apiKey)
	}
}

func TestResolveConfigTimezone(t *testing.T) {
	cfg, err := resolveConfig("db.db", "", "", "America/Chicago", false)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.tzName != "America/Chicago" || cfg.loc == nil {
		t.Errorf("tz = %q loc=%v", cfg.tzName, cfg.loc)
	}
	if _, err := resolveConfig("db.db", "", "", "Not/AZone", false); err == nil {
		t.Error("want error for bad timezone")
	}
}
