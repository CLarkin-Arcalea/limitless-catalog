package catalog

import (
	"sort"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/limitless"
)

// ExtractSpeakers returns the distinct non-empty speaker names in the
// lifelog's blockquote nodes (recursing into children), sorted.
func ExtractSpeakers(l limitless.Lifelog) []string {
	seen := map[string]bool{}
	walkNodes(l.Contents, func(n limitless.ContentNode) {
		if n.Type == "blockquote" && n.SpeakerName != "" {
			seen[n.SpeakerName] = true
		}
	})
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// HasUserSpeaker reports whether any blockquote is attributed to the
// pendant wearer (speakerIdentifier == "user").
func HasUserSpeaker(l limitless.Lifelog) bool {
	found := false
	walkNodes(l.Contents, func(n limitless.ContentNode) {
		if n.Type == "blockquote" && n.SpeakerIdentifier == "user" {
			found = true
		}
	})
	return found
}

func walkNodes(nodes []limitless.ContentNode, fn func(limitless.ContentNode)) {
	for _, n := range nodes {
		fn(n)
		walkNodes(n.Children, fn)
	}
}
