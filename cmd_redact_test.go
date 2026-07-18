package main

import (
	"testing"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/catalog"
	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

func TestCmdRedactRequiresScanSubcommand(t *testing.T) {
	cfg := testCfg(t, "http://unused")
	if err := cmdRedact(cfg, nil); err == nil {
		t.Error("want usage error when no subcommand given")
	}
	if err := cmdRedact(cfg, []string{"bogus"}); err == nil {
		t.Error("want usage error for an unknown subcommand")
	}
}

func TestCmdRedactScanRunsAgainstCatalog(t *testing.T) {
	cfg := testCfg(t, "http://unused")
	s, err := store.Open(cfg.dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if _, err := s.Upsert(catalog.Record{
		ID: "pii1", StartUTC: "2026-07-06T14:00:00Z", EndUTC: "2026-07-06T14:30:00Z",
		LocalDate: "2026-07-06", Title: "budget review", UpdatedAt: "2026-07-06T15:00:00Z",
		Speakers:     []string{"Ava"},
		TranscriptMD: "**Ava:** ssn is 123-45-6789, email chris@example.com",
		Category:     "unknown", RawJSON: "{}",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Upsert(catalog.Record{
		ID: "clean1", StartUTC: "2026-07-07T14:00:00Z", EndUTC: "2026-07-07T14:30:00Z",
		LocalDate: "2026-07-07", Title: "standup", UpdatedAt: "2026-07-07T15:00:00Z",
		Speakers: []string{"Ben"}, TranscriptMD: "**Ben:** nothing sensitive here",
		Category: "unknown", RawJSON: "{}",
	}); err != nil {
		t.Fatal(err)
	}

	if err := cmdRedact(cfg, []string{"scan"}); err != nil {
		t.Fatalf("scan over full catalog: %v", err)
	}
	if err := cmdRedact(cfg, []string{"scan", "--search", "budget"}); err != nil {
		t.Fatalf("scan with --search: %v", err)
	}
	if err := cmdRedact(cfg, []string{"scan", "--start", "2026-07-06", "--end", "2026-07-06"}); err != nil {
		t.Fatalf("scan with date bounds: %v", err)
	}
}

func TestCmdExportRedactSpeakerLeavesCatalogUntouched(t *testing.T) {
	cfg := testCfg(t, "http://unused")
	s, err := store.Open(cfg.dbPath)
	if err != nil {
		t.Fatal(err)
	}
	rec := catalog.Record{
		ID: "r1", StartUTC: "2026-07-06T14:00:00Z", EndUTC: "2026-07-06T14:30:00Z",
		LocalDate: "2026-07-06", Title: "Call with Ava", UpdatedAt: "2026-07-06T15:00:00Z",
		Speakers:     []string{"Ava", "Ben"},
		TranscriptMD: "**Ava:** hi Ben",
		Category:     "unknown", RawJSON: `{"speakerName":"Ava"}`,
	}
	if _, err := s.Upsert(rec); err != nil {
		t.Fatal(err)
	}
	s.Close()

	outDir := t.TempDir()
	if err := cmdExport(cfg, []string{"--format", "md", "--out", outDir, "--redact-speaker", "Ava"}); err != nil {
		t.Fatalf("export: %v", err)
	}

	s2, err := store.Open(cfg.dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	fr, err := s2.Get("r1")
	if err != nil {
		t.Fatal(err)
	}
	if fr == nil {
		t.Fatal("record disappeared")
	}
	if fr.TranscriptMD != rec.TranscriptMD || fr.RawJSON != rec.RawJSON {
		t.Errorf("catalog must not be modified by export redaction, got %+v", fr)
	}
	if len(fr.Speakers) != 2 || fr.Speakers[0] != "Ava" {
		t.Errorf("catalog speakers must not be modified, got %v", fr.Speakers)
	}
}
