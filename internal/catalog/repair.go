package catalog

import (
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/limitless"
)

// SaneFloor is the earliest plausible lifelog date. Real accounts have
// been observed with corrupted epoch-zero startTimes; nothing real
// predates the pendant era, so timestamps before this are treated as
// corrupted and repaired.
const SaneFloor = "2013-01-01"

// saneTime parses s and reports whether it is a usable timestamp on or
// after SaneFloor.
func saneTime(s string) (time.Time, bool) {
	_, t, err := NormalizeUTC(s)
	if err != nil {
		return time.Time{}, false
	}
	if t.UTC().Format("2006-01-02") < SaneFloor {
		return time.Time{}, false
	}
	return t, true
}

// repairStart recovers a usable start time for a lifelog whose top-level
// startTime is corrupted: the earliest sane content-node StartTime wins,
// then the lifelog's EndTime. ok is false when nothing usable exists.
func repairStart(l limitless.Lifelog) (time.Time, bool) {
	var best time.Time
	found := false
	walkNodes(l.Contents, func(n limitless.ContentNode) {
		if t, ok := saneTime(n.StartTime); ok {
			if !found || t.Before(best) {
				best, found = t, true
			}
		}
	})
	if found {
		return best, true
	}
	return saneTime(l.EndTime)
}
