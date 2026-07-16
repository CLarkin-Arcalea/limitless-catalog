// Package limitless is a minimal client for the Limitless Developer API.
package limitless

// ContentNode is one block of a lifelog's structured contents.
type ContentNode struct {
	Type              string        `json:"type"` // heading1|heading2|heading3|blockquote
	Content           string        `json:"content"`
	StartTime         string        `json:"startTime"`
	EndTime           string        `json:"endTime"`
	StartOffsetMs     int64         `json:"startOffsetMs"`
	EndOffsetMs       int64         `json:"endOffsetMs"`
	Children          []ContentNode `json:"children"`
	SpeakerName       string        `json:"speakerName"`
	SpeakerIdentifier string        `json:"speakerIdentifier"` // "user" or ""
}

// Lifelog is one pendant recording as returned by the API.
type Lifelog struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Markdown  string        `json:"markdown"`
	StartTime string        `json:"startTime"`
	EndTime   string        `json:"endTime"`
	IsStarred bool          `json:"isStarred"`
	UpdatedAt string        `json:"updatedAt"`
	Contents  []ContentNode `json:"contents"`
}

type listResponse struct {
	Data struct {
		Lifelogs []Lifelog `json:"lifelogs"`
	} `json:"data"`
	Meta struct {
		Lifelogs struct {
			NextCursor string `json:"nextCursor"`
			Count      int    `json:"count"`
		} `json:"lifelogs"`
	} `json:"meta"`
}
