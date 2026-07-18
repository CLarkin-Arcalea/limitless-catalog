package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

// stringList collects repeated occurrences of a flag into a slice, e.g.
// -redact-speaker Ava -redact-speaker Ben.
type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func cmdExport(cfg config, args []string) error {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	format := fs.String("format", "md", "md or json")
	outDir := fs.String("out", "export", "output directory (md) or file prefix (json)")
	start := fs.String("start", "", "start date YYYY-MM-DD (default: everything)")
	end := fs.String("end", "", "end date YYYY-MM-DD")
	search := fs.String("search", "", "export only conversations matching this phrase")
	var redactSpeakers stringList
	fs.Var(&redactSpeakers, "redact-speaker", "speaker name to redact from exported output (repeatable); does not modify the catalog")
	fs.Parse(args)

	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer s.Close()

	var recs []store.FullRecord
	if *search != "" {
		recs, err = s.ExportRecordsMatching(*search, *start, *end)
	} else {
		recs, err = s.ExportRecords(*start, *end)
	}
	if err != nil {
		return err
	}
	if len(recs) == 0 {
		fmt.Println("nothing to export")
		return nil
	}

	if len(redactSpeakers) > 0 {
		red := newSpeakerRedactor(redactSpeakers)
		for i := range recs {
			recs[i] = red.redact(recs[i])
		}
	}

	switch *format {
	case "md":
		if err := os.MkdirAll(*outDir, 0o755); err != nil {
			return err
		}
		for _, fr := range recs {
			path := filepath.Join(*outDir, exportFilename(fr, cfg.loc))
			if err := os.WriteFile(path, []byte(renderExportMarkdown(fr)), 0o644); err != nil {
				return err
			}
		}
		fmt.Printf("exported %d lifelogs to %s/\n", len(recs), *outDir)
		return nil
	case "json":
		if err := os.MkdirAll(*outDir, 0o755); err != nil {
			return err
		}
		path := filepath.Join(*outDir, "lifelogs.jsonl")
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		for _, fr := range recs {
			if err := enc.Encode(fr); err != nil {
				return err
			}
		}
		fmt.Printf("exported %d lifelogs to %s\n", len(recs), path)
		return nil
	default:
		return fmt.Errorf("unknown format %q (use md or json)", *format)
	}
}

func cmdStats(cfg config, args []string) error {
	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer s.Close()

	st, err := s.Stats(cfg.dbPath)
	if err != nil {
		return err
	}
	if cfg.asJSON {
		out, err := jsonIndent(st)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	fmt.Printf("lifelogs:    %d\n", st.Total)
	if st.Total == 0 {
		fmt.Println("catalog is empty; run: limitless-catalog ingest backfill --days 90")
		return nil
	}
	fmt.Printf("coverage:    %s to %s\n", st.FirstDate, st.LastDate)
	fmt.Printf("hours:       %.1f\n", st.TotalHours)
	fmt.Printf("db size:     %.1f MB\n", float64(st.DBBytes)/(1024*1024))
	fmt.Printf("last ingest: %s\n", st.LastIngest)
	fmt.Println("per month:")
	for _, mc := range st.PerMonth {
		fmt.Printf("  %s  %d\n", mc.Month, mc.Count)
	}
	fmt.Println("by category:")
	for _, cat := range []string{"work", "personal", "media", "unknown"} {
		if n, ok := st.ByCategory[cat]; ok {
			fmt.Printf("  %-9s %d\n", cat, n)
		}
	}
	if n := len(st.EmptyDays); n > 0 {
		show := st.EmptyDays
		if n > 20 {
			show = show[:20]
		}
		fmt.Printf("empty days (%d): %v", n, show)
		if n > 20 {
			fmt.Printf(" and %d more", n-20)
		}
		fmt.Println()
	}
	return nil
}
