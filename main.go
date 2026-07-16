// limitless-catalog: a local, searchable SQLite catalog of your Limitless
// pendant lifelogs. Not affiliated with Limitless.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/limitless"
)

const usageText = `limitless-catalog - local searchable catalog of Limitless lifelogs

Usage:
  limitless-catalog [global flags] <command> [args]

Commands:
  ingest    fetch lifelogs from the Limitless API into the local catalog
  search    full-text search over titles and transcripts
  date      list logs for one local date (YYYY-MM-DD)
  range     list logs for a date range
  recent    list the newest logs
  meeting   find logs overlapping a time window
  get       show one log (use --full for the transcript)
  export    write your archive out as markdown or JSON files
  stats     catalog totals, coverage, and gaps
  span      show the API's oldest/newest lifelogs vs local coverage

Global flags:
  -db string        path to the SQLite catalog (default "limitless.db")
  -api-key string   Limitless API key (default $LIMITLESS_API_KEY)
  -base-url string  API base URL (default production)
  -timezone string  IANA timezone for day bucketing and API fetch windows
                    (default: system local, or UTC if undeterminable)
  -json             emit JSON instead of text
`

func main() {
	dbPath := flag.String("db", "limitless.db", "path to the SQLite catalog")
	apiKey := flag.String("api-key", "", "Limitless API key (default $LIMITLESS_API_KEY)")
	baseURL := flag.String("base-url", limitless.DefaultBaseURL, "API base URL")
	tz := flag.String("timezone", "", "IANA timezone for day bucketing and API fetch windows (default: system local, or UTC if undeterminable)")
	asJSON := flag.Bool("json", false, "emit JSON")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usageText) }
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	cfg, err := resolveConfig(*dbPath, *apiKey, *baseURL, *tz, *asJSON)
	if err != nil {
		fatal(err)
	}

	cmd, rest := flag.Arg(0), flag.Args()[1:]
	switch cmd {
	case "ingest":
		err = cmdIngest(cfg, rest)
	case "search":
		err = cmdSearch(cfg, rest)
	case "date":
		err = cmdDate(cfg, rest)
	case "range":
		err = cmdRange(cfg, rest)
	case "recent":
		err = cmdRecent(cfg, rest)
	case "meeting":
		err = cmdMeeting(cfg, rest)
	case "get":
		err = cmdGet(cfg, rest)
	case "export":
		err = cmdExport(cfg, rest)
	case "stats":
		err = cmdStats(cfg, rest)
	case "span":
		err = cmdSpan(cfg, rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		flag.Usage()
		os.Exit(2)
	}
	if err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
