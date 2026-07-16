package catalog

import "strings"

// mediaMarkers are phrases common in broadcast/podcast/ad audio and rare
// in live conversation. Two or more distinct hits marks a log as media.
var mediaMarkers = []string{
	"subscribe",
	"sponsored by",
	"this episode",
	"we'll be right back",
	"commercial break",
	"terms and conditions",
	"side effects",
	"call now",
	"tune in",
	"streaming now",
}

// Categorize applies a deliberately conservative heuristic. If the pendant
// wearer spoke, it is a real conversation: never media. Otherwise it takes
// two distinct media markers to call it media. Everything else stays
// unknown rather than risk mislabeling. (work/personal come in Phase 2
// via calendar mapping.)
func Categorize(transcript string, hasUser bool) string {
	if hasUser {
		return "unknown"
	}
	lower := strings.ToLower(transcript)
	hits := 0
	for _, m := range mediaMarkers {
		if strings.Contains(lower, m) {
			hits++
		}
	}
	if hits >= 2 {
		return "media"
	}
	return "unknown"
}
