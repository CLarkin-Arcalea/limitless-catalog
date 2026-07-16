CREATE TABLE IF NOT EXISTS lifelogs (
  id                   TEXT PRIMARY KEY,
  start_utc            TEXT NOT NULL,
  end_utc              TEXT NOT NULL,
  local_date           TEXT NOT NULL,
  title                TEXT NOT NULL DEFAULT '',
  duration_min         REAL NOT NULL DEFAULT 0,
  is_starred           INTEGER NOT NULL DEFAULT 0,
  updated_at           TEXT NOT NULL DEFAULT '',
  speakers             TEXT NOT NULL DEFAULT '[]',
  transcript_md        TEXT NOT NULL DEFAULT '',
  category             TEXT NOT NULL DEFAULT 'unknown',
  calendar_event_id    TEXT,
  calendar_event_title TEXT,
  ingested_at          TEXT NOT NULL,
  raw_json             TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_lifelogs_local_date ON lifelogs(local_date);
CREATE INDEX IF NOT EXISTS idx_lifelogs_start_utc  ON lifelogs(start_utc);

CREATE VIRTUAL TABLE IF NOT EXISTS lifelogs_fts USING fts5(
  title,
  transcript_md,
  id UNINDEXED,
  tokenize = 'porter unicode61'
);

CREATE TABLE IF NOT EXISTS ingest_state (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

PRAGMA user_version = 1;
