package store

import (
	"path/filepath"
	"testing"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenInitializesSchema(t *testing.T) {
	s := openTemp(t)

	for _, table := range []string{"lifelogs", "lifelogs_fts", "ingest_state"} {
		var name string
		err := s.DB().QueryRow(
			`SELECT name FROM sqlite_master WHERE name = ?`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %s missing: %v", table, err)
		}
	}

	var version int
	if err := s.DB().QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatalf("user_version: %v", err)
	}
	if version != 1 {
		t.Errorf("user_version = %d, want 1", version)
	}

	var mode string
	if err := s.DB().QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s1, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	s1.Close()
	s2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	s2.Close()
}

func TestOpenReadOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ro.db")
	s, err := Open(path) // create + init
	if err != nil {
		t.Fatal(err)
	}
	s.Close()

	ro, err := OpenReadOnly(path)
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	defer ro.Close()

	if _, err := ro.DB().Exec(`INSERT INTO ingest_state (key, value) VALUES ('x','y')`); err == nil {
		t.Error("write on read-only handle must fail")
	}
	var n int
	if err := ro.DB().QueryRow(`SELECT COUNT(*) FROM lifelogs`).Scan(&n); err != nil {
		t.Errorf("read on read-only handle: %v", err)
	}
}

func TestOpenReadOnlyMissingDB(t *testing.T) {
	if _, err := OpenReadOnly(filepath.Join(t.TempDir(), "absent.db")); err == nil {
		t.Fatal("want error for missing catalog")
	}
}
