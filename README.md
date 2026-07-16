# limitless-catalog

A single-binary tool that backs up your [Limitless](https://www.limitless.ai)
pendant lifelogs into a local, searchable SQLite catalog you own.

Your recordings, your transcripts, your machine. Fast full-text search,
date and meeting-window lookups, and full export to plain markdown or JSON.

Not affiliated with Limitless. Built by and for pendant owners who want a
durable local archive of their own data.

## Why

Live API access is fine until it isn't: payloads are large, range queries
are awkward, and if the service ever winds down, your history goes with it.
This tool pulls everything once, keeps it updated incrementally, and makes
it instantly queryable offline. The export command turns the whole archive
into plain files any tool can read.

## Install

With Go 1.25+:

    go install github.com/CLarkin-Arcalea/limitless-catalog@latest

Or build from a clone:

    go build -o limitless-catalog .

Or cross-compile for any platform (pure Go, no C toolchain needed):

    GOOS=linux  GOARCH=amd64 go build -o dist/limitless-catalog-linux-amd64 .
    GOOS=darwin GOARCH=arm64 go build -o dist/limitless-catalog-darwin-arm64 .
    GOOS=windows GOARCH=amd64 go build -o dist/limitless-catalog-windows-amd64.exe .

## Setup

1. Get your API key from the Limitless app: Settings, Developer, create key.
2. Export it:

       export LIMITLESS_API_KEY=your-key-here

   (Or pass `--api-key` per command.)

## Usage

See what your account has before downloading:

    limitless-catalog span

First backfill (pick a window):

    limitless-catalog ingest backfill --months 3        # last 3 months
    limitless-catalog ingest backfill --all             # everything
    limitless-catalog ingest backfill --from-start --months 6   # first 6 months of your history
    limitless-catalog ingest backfill --days 90         # or by days
    limitless-catalog ingest backfill --start 2026-01-01 --end 2026-03-31

Keep it current (run this daily; see Scheduling below):

    limitless-catalog ingest incremental

Query:

    limitless-catalog search "kubernetes migration"
    limitless-catalog date 2026-07-06
    limitless-catalog range 2026-07-01 2026-07-06
    limitless-catalog recent 10
    limitless-catalog meeting --end "2026-07-06 14:00" "2026-07-06 13:00"
    limitless-catalog get --full <id>

Every query takes the global `-json` flag for machine-readable output; like
all global flags, it goes before the subcommand (for example:
`limitless-catalog -json recent 5`).

Own your data:

    limitless-catalog export --format md --out export/
    limitless-catalog export --format json
    limitless-catalog stats

`stats` shows coverage and any days with zero recordings, so you can spot
gaps in a backfill.

## Global flags

    -db string        catalog path (default "limitless.db")
    -api-key string   API key (default $LIMITLESS_API_KEY)
    -timezone string  IANA timezone for day bucketing and API fetch windows
                      (default: system local, or UTC if undeterminable)
    -json             JSON output

## Scheduling

macOS launchd (`~/Library/LaunchAgents/com.limitless-catalog.plist`):

    <?xml version="1.0" encoding="UTF-8"?>
    <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
      "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
    <plist version="1.0"><dict>
      <key>Label</key><string>com.limitless-catalog</string>
      <key>ProgramArguments</key><array>
        <string>/usr/local/bin/limitless-catalog</string>
        <string>ingest</string><string>incremental</string>
      </array>
      <key>EnvironmentVariables</key><dict>
        <key>LIMITLESS_API_KEY</key><string>your-key-here</string>
      </dict>
      <key>StartCalendarInterval</key><dict>
        <key>Hour</key><integer>6</integer><key>Minute</key><integer>30</integer>
      </dict>
      <key>WorkingDirectory</key><string>/path/to/your/catalog</string>
    </dict></plist>

Linux cron:

    30 6 * * * cd /path/to/catalog && LIMITLESS_API_KEY=your-key ./limitless-catalog ingest incremental

## Using the catalog from other tools

The database is plain SQLite with an FTS5 index; query it from anything:

    sqlite3 limitless.db "SELECT local_date, title FROM lifelogs ORDER BY start_utc DESC LIMIT 5"

Example: an assistant workflow that debriefs a 1pm meeting can run
`limitless-catalog -json meeting "2026-07-06 13:00"`, take the returned
ids, and `limitless-catalog get --full <id>` for the transcript.

## Notes

- The pendant records background media (TV, podcasts). A conservative
  heuristic tags obvious cases as `media`; everything else stays `unknown`.
- Re-running ingest is always safe: records are deduped by id and only
  rewritten when the API's `updatedAt` changes.
- Respect the API rate limit (180 requests/minute); the client honors
  `Retry-After` and backs off automatically.
- Some accounts contain corrupted lifelogs with epoch-zero timestamps; the
  oldest-lifelog anchor used by span and backfill --all/--from-start ignores
  anything before 2013 and refetches from there.

## License

MIT
