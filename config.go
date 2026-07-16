package main

import (
	"fmt"
	"os"
	"time"
)

type config struct {
	dbPath  string
	apiKey  string
	baseURL string
	loc     *time.Location
	tzName  string
	asJSON  bool
}

// resolveConfig applies precedence: flags beat environment beats defaults.
// tzFlag empty means system local time.
func resolveConfig(dbPath, apiKeyFlag, baseURL, tzFlag string, asJSON bool) (config, error) {
	apiKey := apiKeyFlag
	if apiKey == "" {
		apiKey = os.Getenv("LIMITLESS_API_KEY")
	}

	loc, tzName, err := resolveTZ(tzFlag, time.Local)
	if err != nil {
		return config{}, err
	}

	return config{
		dbPath:  dbPath,
		apiKey:  apiKey,
		baseURL: baseURL,
		loc:     loc,
		tzName:  tzName,
		asJSON:  asJSON,
	}, nil
}

// resolveTZ picks the timezone used for BOTH the API's day-window
// parameter and local_date bucketing. They must always agree: fetching
// one day span and bucketing/marking-done in a different one can
// silently and permanently skip edge-of-day lifelogs.
func resolveTZ(tzFlag string, localLoc *time.Location) (*time.Location, string, error) {
	if tzFlag != "" {
		l, err := time.LoadLocation(tzFlag)
		if err != nil {
			return nil, "", fmt.Errorf("unknown timezone %q (use an IANA name like America/Chicago): %w", tzFlag, err)
		}
		return l, tzFlag, nil
	}
	name := localLoc.String()
	// "Local" is not an IANA name the API understands. When the real
	// zone name is unknown, fall back to UTC for BOTH sides so fetch
	// windows and stored dates always agree.
	if name == "Local" {
		return time.UTC, "UTC", nil
	}
	return localLoc, name, nil
}
