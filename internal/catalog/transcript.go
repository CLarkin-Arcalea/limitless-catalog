package catalog

import (
	"fmt"
	"strings"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/limitless"
)

// AssembleTranscript returns the lifelog's transcript as markdown.
// The API's own markdown field wins when present; otherwise the transcript
// is rebuilt from the structured contents tree.
func AssembleTranscript(l limitless.Lifelog) string {
	if strings.TrimSpace(l.Markdown) != "" {
		return l.Markdown
	}
	var b strings.Builder
	writeNodes(&b, l.Contents)
	return strings.TrimSuffix(b.String(), "\n\n")
}

func writeNodes(b *strings.Builder, nodes []limitless.ContentNode) {
	for _, n := range nodes {
		switch n.Type {
		case "heading1":
			fmt.Fprintf(b, "# %s\n\n", n.Content)
		case "heading2":
			fmt.Fprintf(b, "## %s\n\n", n.Content)
		case "heading3":
			fmt.Fprintf(b, "### %s\n\n", n.Content)
		case "blockquote":
			name := n.SpeakerName
			if name == "" {
				name = "Unknown"
			}
			fmt.Fprintf(b, "**%s:** %s\n\n", name, n.Content)
		default:
			if n.Content != "" {
				fmt.Fprintf(b, "%s\n\n", n.Content)
			}
		}
		writeNodes(b, n.Children)
	}
}
