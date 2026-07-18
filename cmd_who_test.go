package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/catalog"
	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

func seedWhoDB(t *testing.T, dbPath string) {
	t.Helper()
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	recs := []catalog.Record{
		{ID: "a", StartUTC: "2026-07-05T14:00:00Z", EndUTC: "2026-07-05T14:30:00Z",
			LocalDate: "2026-07-05", Title: "AMS quoting call", DurationMin: 30,
			UpdatedAt: "u1", Speakers: []string{"Ava", "Mike"},
			TranscriptMD: "**Ava:** hi", Category: "unknown", RawJSON: `{"id":"a"}`},
		{ID: "b", StartUTC: "2026-07-06T18:00:00Z", EndUTC: "2026-07-06T18:45:00Z",
			LocalDate: "2026-07-06", Title: "Ben 1:1", DurationMin: 45,
			UpdatedAt: "u2", Speakers: []string{"Ava", "Ben"},
			TranscriptMD: "**Ben:** hi", Category: "unknown", RawJSON: `{"id":"b"}`},
	}
	for _, r := range recs {
		if _, err := s.Upsert(r); err != nil {
			t.Fatalf("seed %s: %v", r.ID, err)
		}
	}
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	fnErr := fn()
	w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String(), fnErr
}

func TestCmdWhoRejectsBadArgs(t *testing.T) {
	cfg := config{dbPath: "unused.db"}
	if err := cmdWho(cfg, []string{"Ava", "extra"}); err == nil {
		t.Error("cmdWho with 2 args: want usage error, got nil")
	}
}

func TestCmdWhoUnknownSpeaker(t *testing.T) {
	cfg := testCfg(t, "")
	seedWhoDB(t, cfg.dbPath)

	if err := cmdWho(cfg, []string{"Nobody"}); err == nil {
		t.Error("want error for unknown speaker")
	}
}

func TestCmdWhoListText(t *testing.T) {
	cfg := testCfg(t, "")
	seedWhoDB(t, cfg.dbPath)

	out, err := captureStdout(t, func() error { return cmdWho(cfg, nil) })
	if err != nil {
		t.Fatalf("cmdWho: %v", err)
	}
	avaIdx := strings.Index(out, "Ava")
	benIdx := strings.Index(out, "Ben")
	if avaIdx == -1 || benIdx == -1 {
		t.Fatalf("output missing expected speakers:\n%s", out)
	}
	if avaIdx > benIdx {
		t.Errorf("want Ava (count 2) ranked before Ben (count 1):\n%s", out)
	}
	if !strings.Contains(out, "2  Ava") {
		t.Errorf("want Ava's count of 2 in output:\n%s", out)
	}
}

func TestCmdWhoDetailJSON(t *testing.T) {
	cfg := testCfg(t, "")
	cfg.asJSON = true
	seedWhoDB(t, cfg.dbPath)

	out, err := captureStdout(t, func() error { return cmdWho(cfg, []string{"Ava"}) })
	if err != nil {
		t.Fatalf("cmdWho: %v", err)
	}
	for _, want := range []string{`"name": "Ava"`, `"count": 2`, `"first_seen": "2026-07-05"`,
		`"last_seen": "2026-07-06"`, `"longest_gap_days": 1`} {
		if !strings.Contains(out, want) {
			t.Errorf("json output missing %q in:\n%s", want, out)
		}
	}
}

func TestCmdWhoDetailText(t *testing.T) {
	cfg := testCfg(t, "")
	seedWhoDB(t, cfg.dbPath)

	out, err := captureStdout(t, func() error { return cmdWho(cfg, []string{"Mike"}) })
	if err != nil {
		t.Fatalf("cmdWho: %v", err)
	}
	for _, want := range []string{"Mike", "lifelogs:        1", "first seen:      2026-07-05",
		"last seen:       2026-07-05", "days since last: 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q in:\n%s", want, out)
		}
	}
}
