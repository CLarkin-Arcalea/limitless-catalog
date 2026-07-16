package main

import (
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/store"
)

func openStore(cfg config) (*store.Store, error) {
	return store.Open(cfg.dbPath)
}

func cmdSearch(cfg config, args []string) error {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	limit := fs.Int("limit", 20, "max results")
	fs.Parse(args)
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: search [--limit N] <term>")
	}
	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer s.Close()
	rows, err := s.Search(fs.Arg(0), *limit)
	if err != nil {
		return err
	}
	return printRows(rows, cfg.asJSON)
}

func cmdDate(cfg config, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: date <YYYY-MM-DD>")
	}
	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer s.Close()
	rows, err := s.ByDate(args[0])
	if err != nil {
		return err
	}
	return printRows(rows, cfg.asJSON)
}

func cmdRange(cfg config, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: range <start YYYY-MM-DD> <end YYYY-MM-DD>")
	}
	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer s.Close()
	rows, err := s.ByRange(args[0], args[1])
	if err != nil {
		return err
	}
	return printRows(rows, cfg.asJSON)
}

func cmdRecent(cfg config, args []string) error {
	n := 10
	if len(args) > 1 {
		return fmt.Errorf("usage: recent [n] (global flags like -json go before the subcommand)")
	}
	if len(args) == 1 {
		v, err := strconv.Atoi(args[0])
		if err != nil || v < 1 {
			return fmt.Errorf("usage: recent [n] (n must be a positive integer)")
		}
		n = v
	}
	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer s.Close()
	rows, err := s.Recent(n)
	if err != nil {
		return err
	}
	return printRows(rows, cfg.asJSON)
}

func cmdMeeting(cfg config, args []string) error {
	fs := flag.NewFlagSet("meeting", flag.ExitOnError)
	end := fs.String("end", "", "meeting end (defaults to start + 1h)")
	buffer := fs.Int("buffer", 10, "minutes of slack on each side")
	fs.Parse(args)
	if fs.NArg() != 1 {
		return fmt.Errorf(`usage: meeting [--end "YYYY-MM-DD HH:MM"] [--buffer 10] "<YYYY-MM-DD HH:MM>"`)
	}
	startT, err := parseLocalDateTime(fs.Arg(0), cfg.loc)
	if err != nil {
		return err
	}
	endT := startT.Add(time.Hour)
	if *end != "" {
		endT, err = parseLocalDateTime(*end, cfg.loc)
		if err != nil {
			return err
		}
	}
	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer s.Close()
	rows, err := s.Meeting(startT, endT, time.Duration(*buffer)*time.Minute)
	if err != nil {
		return err
	}
	return printRows(rows, cfg.asJSON)
}

func cmdGet(cfg config, args []string) error {
	fs := flag.NewFlagSet("get", flag.ExitOnError)
	full := fs.Bool("full", false, "print the full transcript")
	fs.Parse(args)
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: get [--full] <id>")
	}
	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer s.Close()
	fr, err := s.Get(fs.Arg(0))
	if err != nil {
		return err
	}
	if fr == nil {
		return fmt.Errorf("no lifelog with id %q", fs.Arg(0))
	}
	if cfg.asJSON {
		if !*full {
			fr.TranscriptMD = ""
		}
		fr.RawJSON = "" // raw stays opt-in via export
		b, err := jsonIndent(fr)
		if err != nil {
			return err
		}
		fmt.Println(b)
		return nil
	}
	out, err := formatRows([]store.Row{fr.Row}, false)
	if err != nil {
		return err
	}
	fmt.Print(out)
	if *full {
		fmt.Println("\n" + fr.TranscriptMD)
	}
	return nil
}

func printRows(rows []store.Row, asJSON bool) error {
	out, err := formatRows(rows, asJSON)
	if err != nil {
		return err
	}
	fmt.Print(out)
	if asJSON {
		fmt.Println()
	}
	return nil
}
