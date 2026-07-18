package main

import (
	"flag"
	"fmt"
)

func cmdWho(cfg config, args []string) error {
	fs := flag.NewFlagSet("who", flag.ExitOnError)
	fs.Parse(args)
	if fs.NArg() > 1 {
		return fmt.Errorf("usage: who [name]")
	}

	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer s.Close()

	if fs.NArg() == 1 {
		st, err := s.Speaker(fs.Arg(0))
		if err != nil {
			return err
		}
		if st == nil {
			return fmt.Errorf("no lifelogs with speaker %q", fs.Arg(0))
		}
		if cfg.asJSON {
			out, err := jsonIndent(st)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		}
		fmt.Printf("%s\n", st.Name)
		fmt.Printf("  lifelogs:        %d\n", st.Count)
		fmt.Printf("  first seen:      %s\n", st.FirstSeen)
		fmt.Printf("  last seen:       %s\n", st.LastSeen)
		fmt.Printf("  days since last: %d\n", st.DaysSinceLast)
		fmt.Printf("  longest gap:     %d days\n", st.LongestGapDays)
		return nil
	}

	stats, err := s.Speakers()
	if err != nil {
		return err
	}
	if cfg.asJSON {
		out, err := jsonIndent(stats)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}
	if len(stats) == 0 {
		fmt.Println("no speakers found")
		return nil
	}
	for _, st := range stats {
		fmt.Printf("%4d  %-25s  first %s  last %s  (%dd ago)\n",
			st.Count, st.Name, st.FirstSeen, st.LastSeen, st.DaysSinceLast)
	}
	return nil
}
