# Changelog

## v1.1.0 (unreleased)

### Added
- `mcp` subcommand: serve the catalog to MCP clients (Claude Code, Claude Desktop, and others) over stdio, read-only. Seven tools: `search_transcripts`, `find_meeting`, `list_by_date`, `list_range`, `recent`, `get_transcript`, `catalog_stats`. The database is opened in read-only mode; no tool can write or trigger network fetches.
- `ingest orphans`: rescue lifelogs with corrupted (epoch-era) timestamps that no date query can reach. Real start times are recovered from the transcript's own line timestamps where possible, falling back to the record's end time.
- `ingest --id <lifelog-id>`: fetch and ingest a single lifelog by id.
- `export --search "<phrase>"`: export only the conversations matching a full-text query, combinable with `--start`/`--end` and both output formats.

### Fixed
- Records with corrupted timestamps are now repaired on ingest instead of stored with epoch-era dates (applies to both start and end times).

### Changed
- Second dependency added: `github.com/modelcontextprotocol/go-sdk` (official MCP SDK, pure Go, keeps cross-compilation cgo-free).

## v1.0.0 (2026-07-16)

Initial public release: resumable backfill with window flags (`--days`, `--months`, `--all`, `--from-start`, `--start`/`--end`), incremental ingest, `span` coverage discovery, full-text `search`, date and meeting-window queries, `get` transcripts, markdown/JSONL `export`, coverage `stats` with gap detection. Single static binary, SQLite + FTS5, MIT.
