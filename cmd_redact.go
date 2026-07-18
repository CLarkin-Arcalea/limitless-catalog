package main

import (
	"flag"
	"fmt"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

const heuristicNotice = "heuristic scan (simple regexp shape-matching); not a compliance guarantee"

// cmdRedact handles: redact scan [--search TERM] [--start DATE] [--end DATE]
func cmdRedact(cfg config, args []string) error {
	if len(args) == 0 || args[0] != "scan" {
		return fmt.Errorf("usage: redact scan [--search TERM] [--start DATE] [--end DATE]")
	}
	args = args[1:]

	fs := flag.NewFlagSet("redact scan", flag.ExitOnError)
	search := fs.String("search", "", "scan only conversations matching this phrase")
	start := fs.String("start", "", "start date YYYY-MM-DD (default: everything)")
	end := fs.String("end", "", "end date YYYY-MM-DD")
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

	var findings []piiFinding
	for _, fr := range recs {
		findings = append(findings, scanForPII(fr)...)
	}

	if cfg.asJSON {
		out, err := jsonIndent(map[string]any{
			"notice":   heuristicNotice,
			"findings": findings,
		})
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	fmt.Println(heuristicNotice)
	if len(findings) == 0 {
		fmt.Println("no likely PII patterns found")
		return nil
	}
	fmt.Printf("%d potential match(es); matched text is never shown here, only the pattern kind and location:\n", len(findings))
	for _, f := range findings {
		fmt.Printf("  %s  %-12s  %s\n", f.LocalDate, f.Kind, f.LifelogID)
	}
	return nil
}
