# Architecture: limitless-catalog

This is the design and architecture reference for contributors. It records why the
tool exists, how the pieces fit together, and the decisions (with rationale) that
shape the codebase. Read this before making structural changes.

---

## 1. Problem & Goal

Pulling pendant transcripts live through intermediary integrations (such as the
Limitless MCP server) is unreliable and slow in practice:

- Range queries can return the wrong logs (a morning query returning an afternoon log).
- Payloads are huge and truncate (a full-day list can collapse to a single item).
- Every use requires saving the response somewhere and re-parsing it.

**Goal:** a local, indexed SQLite catalog of Limitless lifelogs so transcripts are
fast, correct, and reusable. Built against the **Limitless Developer API directly**
(not the MCP), it becomes a durable local archive the user owns.

**Mission (why open source):** help people keep access to their own Limitless data,
including if the service ever winds down. The tool is built for strangers to
download, run, and contribute to, not for any single operator.

---

## 2. Core decisions

| # | Decision | Choice | Rationale |
|---|----------|--------|-----------|
| 1 | Language | **Go** | Single static cross-platform binary via `modernc.org/sqlite` (pure Go, no cgo, FTS5 included). Trivial cross-compilation for prebuilt binaries. Readable by outside contributors. Fast path to v1. |
| 2 | License | **MIT** | Maximally permissive so anyone can reuse it. |
| 3 | Scope | **Phase 1 only** | `lifelogs` + FTS5 + ingest (backfill/incremental) + query CLI + `export` + `stats`. A line-level `segments` table and calendar-event mapping are deferred. The schema keeps nullable calendar columns so Phase 2 needs no migration. |
| 4 | First backfill depth | **90 days default**, `--days` flag | Balance history vs first-run time and API call volume. |
| 5 | Name | `limitless-catalog` | Descriptive and discoverable. The README notes it is not affiliated with Limitless. |

**Future work (out of scope now):** (a) the `segments` line-level table; (b) calendar
event mapping; (c) a documented example integration showing how an assistant workflow
could query the DB.

---

## 3. Dependencies

Deliberately minimal:

- **Standard library** for everything possible: `net/http` (API client),
  `encoding/json`, `time` + `time.LoadLocation` (timezone-to-date bucketing),
  `flag` + `flag.FlagSet` (subcommand CLI, no external CLI framework), `embed`
  (embed `schema.sql`).
- **`modernc.org/sqlite`** is the only third-party dependency. Pure-Go SQLite with
  FTS5, registered as a `database/sql` driver. Keeps cross-compilation cgo-free.

No web framework, no CLI framework, no ORM. The dependency surface is one module.
Keep it that way unless there is a very strong reason not to.

---

## 4. Limitless Developer API (verify against live docs before changing the client)

**Re-verify against the current Limitless API docs whenever touching the client**;
stored knowledge and third-party writeups go stale. The shape the client is built
against:

- Endpoint: `GET https://api.limitless.ai/v1/lifelogs`
- Auth header: `X-API-Key: <key>`
- Params: `date` (YYYY-MM-DD) **or** `start`/`end`; `timezone`; `cursor`; `limit`
  (max **10** per request); `includeMarkdown` (bool); `includeHeadings` (bool);
  `direction` (`asc`/`desc`).
- Response: `data.lifelogs[]` plus `meta.lifelogs.nextCursor` for pagination.
- Each lifelog: `id`, `title`, `markdown`, `startTime`, `endTime`, `isStarred`,
  `updatedAt`, `contents[]` blocks (`type` = heading1/2/3 or blockquote; blockquote
  blocks carry `speakerName`, `speakerIdentifier` [user vs other], `content` text,
  per-line timestamps).

The API client and response structs are the only things that change if the live
shape differs. The schema and query layer are insulated behind them.

---

## 5. Configuration

- **API key:** `LIMITLESS_API_KEY` environment variable (primary). Never committed,
  never hardcoded. `--api-key` flag override for scripting.
- **DB path:** `--db` flag, default `./limitless.db`.
- **Timezone:** `--timezone` flag, default to the system local timezone (not
  hardcoded to any region). Controls `local_date` bucketing.
- **Base URL:** `--base-url` flag, default the production endpoint (lets tests and
  contributors point at a mock server).

---

## 6. Data model (`schema.sql`, embedded via `//go:embed`)

### `lifelogs`
| Column | Type | Notes |
|---|---|---|
| `id` | TEXT PRIMARY KEY | Limitless lifelog id; dedup key |
| `start_utc` | TEXT | ISO-8601 UTC |
| `end_utc` | TEXT | ISO-8601 UTC |
| `local_date` | TEXT | `YYYY-MM-DD`, bucketed in the configured timezone |
| `title` | TEXT | |
| `duration_min` | REAL | derived from start/end |
| `is_starred` | INTEGER | 0/1 |
| `updated_at` | TEXT | from API; drives re-ingest |
| `speakers` | TEXT | JSON array of distinct speaker names |
| `transcript_md` | TEXT | assembled full text |
| `category` | TEXT | `work` \| `personal` \| `media` \| `unknown` |
| `calendar_event_id` | TEXT NULL | Phase 2 |
| `calendar_event_title` | TEXT NULL | Phase 2 |
| `ingested_at` | TEXT | when row written |
| `raw_json` | TEXT | full API object, to reprocess without re-fetch |

### `lifelogs_fts` (FTS5)
Standalone (non-external-content) FTS5 table over `title` and `transcript_md`, with
`id` stored UNINDEXED. Kept in sync on upsert: `DELETE FROM lifelogs_fts WHERE id=?`
then `INSERT`. Standalone was chosen deliberately because `lifelogs.id` is a TEXT
primary key, so external-content rowid coupling would add trigger complexity for no
benefit. Tokenizer: `porter unicode61`.

### `ingest_state`
`key TEXT PRIMARY KEY, value TEXT`. Tracks incremental progress (e.g.
`last_incremental_run`) so incremental mode knows where to resume.

### `segments` (DEFERRED)
Documented shape only: `lifelog_id, seq, speaker, text, start_utc`. Added later if
line-level queries are wanted.

### Schema hardening
- Indexes on `local_date` and `start_utc` (every query path hits one of these).
- `PRAGMA user_version = 1` for schema versioning, so future releases can migrate
  existing users' DBs.
- Open with WAL mode + a busy timeout, so a scheduled ingest and an interactive
  query never collide.
- **Timestamp normalization:** all stored timestamps are RFC3339 UTC with `Z`
  suffix and consistent precision, so lexicographic comparison in SQL is always
  correct for range/overlap queries.

---

## 7. Package layout

```
limitless-catalog/
  go.mod                        module github.com/CLarkin-Arcalea/limitless-catalog
  main.go                       subcommand dispatch (ingest | search | date | range | recent | meeting | onthisday | get | export | stats | span)
  README.md                     user-facing setup and usage
  LICENSE                       MIT
  .gitignore                    limitless.db, build artifacts
  internal/
    limitless/                  API client: fetch + pagination + response structs
    store/                      sqlite open/init (embedded schema.sql), upsert, FTS sync, query functions
    catalog/                    assembly: transcript_md, speakers, duration, category heuristic
  internal/limitless/*_test.go  client tests against a mock server (--base-url)
  internal/catalog/*_test.go    assembly + category tests on fixture lifelogs
  internal/store/*_test.go      store tests on temp DBs
```

Single binary with subcommands (idiomatic Go, one artifact to distribute).

---

## 8. Ingestion (`ingest` subcommand)

**Modes:**
- `ingest backfill [--days 90] [--start DATE --end DATE]` loops dates; per date it
  paginates `limit=10` with `cursor` until `nextCursor` is empty.
- `ingest incremental` runs from `max(local_date)` (or `ingest_state`) forward to
  today; idempotent.
- `ingest --date YYYY-MM-DD` handles a single day.

**Per lifelog:**
1. Assemble `transcript_md`: prefer the API `markdown` field; else build from
   `contents[]` (headings as `#`/`##`, blockquotes as `**Speaker:** text`).
2. Extract distinct `speakers` (JSON array) from blockquote blocks.
3. Compute `duration_min`, `local_date` (configured tz via `time.LoadLocation`),
   `is_starred`.
4. **Category heuristic (cheap first pass):** if speakers are all unknown/none and
   the text shows media markers (ad copy, show-dialogue patterns), tag `media`;
   otherwise `unknown`. Deliberately conservative so real conversations stay
   `unknown` rather than being mislabeled. `work`/`personal` refinement can come
   later (e.g. via Phase 2 calendar mapping).
5. **Upsert by `id`.** If the row exists and stored `updated_at` equals the API
   `updatedAt`, skip. If `updatedAt` changed, rewrite the row + FTS. Fully
   idempotent.

**Today is never done:** a backfill or incremental run never marks the current day
as complete in `ingest_state`, because more lifelogs can still arrive for it. Only
fully elapsed days are recorded as complete.

**Timezone coherence:** the same configured timezone drives both the API fetch
windows and `local_date` bucketing, so a day's fetch and a day's bucket always
agree.

**Operational:** safe to re-run; runnable daily via launchd/cron (the README ships
a sample entry). Respects the 10-item request limit and cursor pagination.

**Rate limiting (a 90-day backfill is ~90+ requests minimum, so this matters):**
- Verify the documented rate limits against live API docs when touching the client.
- Honor `Retry-After` on HTTP 429; exponential backoff with cap on 5xx.
- Polite default pacing between requests.
- Backfill records per-date completion in `ingest_state`, so an interrupted
  backfill resumes without re-fetching completed dates (upsert already makes
  re-fetch *safe*; skip-fetch makes it *cheap*).

---

## 9. Query subcommands (also the reusable `store` query functions)

Return concise metadata + snippets, never full dumps, except an explicit
single-record full fetch.

- `search "<term>"` returns FTS matches: `id, local_date, title, speakers, snippet`.
- `date <YYYY-MM-DD>` / `range <start> <end>` return metadata rows for the window.
- `recent [n]` returns the n newest logs (metadata only).
- `meeting "<start>" [--end "<end>"] [--buffer 10]` returns lifelogs **overlapping**
  a time window (default 10-minute buffer each side). Overlap, not containment, so
  it catches recordings that start before or end after the given bounds. This is
  the query a meeting-debrief workflow would use.
- `onthisday [--date YYYY-MM-DD]` returns lifelogs sharing today's (or the
  given date's) month and day in any prior year, newest year first. The
  personal-archive equivalent of photo-app "memories."
- `get <id> [--full]` returns one record; `--full` prints the whole `transcript_md`.

Output default is human-readable text; a `-json` global flag emits JSON so other
tools can consume it.

### `export`: the mission-critical subcommand
The point of this tool is that people keep their own data. A SQLite file is a
better index, but plain files are the escape hatch a non-technical user actually
needs if the service goes dark.

- `export --format md [--out DIR] [--start DATE --end DATE]` writes one markdown
  file per lifelog, named `YYYY-MM-DD-HHMM-<slugified-title>-<id-prefix>.md`, with
  a small metadata header (title, times, speakers, category, id).
- `export --format json` writes the same selection as newline-delimited JSON (full
  records including `raw_json`).
- Defaults to the full archive; date flags narrow it.

### `span`: discover before you download
Shows the oldest and newest lifelogs available on the API (one ascending and one
descending request, no date filter), next to local DB coverage, so the user sees
exactly what is missing at either end and gets the suggested backfill command.

Backfill window flags to go with it:
- `--months N` runs from now back N months.
- `--from-start --months N` runs from the oldest available forward N months.
- `--all` runs from the oldest available through today.
All windows run through the same per-date resume machinery as `--days`.

### `stats`: trust and gap detection
After a backfill, the user needs to know it worked:
- Total lifelogs, date range covered, total recorded hours.
- Per-month counts, plus days in range with **zero** logs (gap candidates: missed
  ingest vs. pendant off).
- Category breakdown, DB file size, last ingest run.

---

## 10. Testing

- **`internal/limitless`**: the client hits an `httptest` mock server (via
  `--base-url`), covering pagination (multi-page `nextCursor`), the 10-item cap,
  and 429/5xx backoff.
- **`internal/catalog`**: fixture lifelog JSON, asserting assembled
  `transcript_md`, extracted speakers, duration, and category outcomes.
- **`internal/store`**: open a temp-file DB, assert upsert idempotency (same
  `updatedAt` skips, changed `updatedAt` rewrites), FTS search returns expected
  ids, and query functions.
- No network in tests. All API interaction goes through the mock.

---

## 11. Gotchas (from hard experience)

- Intermediary range/list queries (MCP-style) are unreliable and truncate. That is
  the reason this exists. Use the direct API.
- Lifelogs are large. Assemble and store the transcript once at ingest; never
  re-fetch to read.
- The pendant captures a lot of background media (TV, radio, podcasts). The
  `category` tag matters for filtering noise.
- Dedup by `id`; honor `updatedAt` for re-ingest.
- Some accounts contain corrupted lifelogs with epoch-zero timestamps; the
  oldest-lifelog anchor used by `span` and `backfill --all/--from-start` ignores
  anything before 2013.
