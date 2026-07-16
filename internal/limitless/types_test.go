package limitless

import (
	"encoding/json"
	"testing"
)

const sampleResponse = `{
  "data": {
    "lifelogs": [
      {
        "id": "abc123",
        "title": "Morning standup",
        "markdown": "# Morning standup\n\nAva: hello",
        "startTime": "2026-07-06T14:00:00.000Z",
        "endTime": "2026-07-06T14:30:00.000Z",
        "isStarred": true,
        "updatedAt": "2026-07-06T15:00:00.000Z",
        "contents": [
          {
            "type": "heading1",
            "content": "Morning standup",
            "startTime": "2026-07-06T14:00:00.000Z",
            "endTime": "2026-07-06T14:30:00.000Z",
            "startOffsetMs": 0,
            "endOffsetMs": 1800000,
            "children": [
              {
                "type": "blockquote",
                "content": "hello",
                "speakerName": "Ava",
                "speakerIdentifier": "user",
                "children": []
              }
            ],
            "speakerName": null,
            "speakerIdentifier": null
          }
        ]
      }
    ]
  },
  "meta": { "lifelogs": { "nextCursor": "cur_2", "count": 1 } }
}`

func TestDecodeListResponse(t *testing.T) {
	var resp listResponse
	if err := json.Unmarshal([]byte(sampleResponse), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	logs := resp.Data.Lifelogs
	if len(logs) != 1 {
		t.Fatalf("got %d lifelogs, want 1", len(logs))
	}
	l := logs[0]
	if l.ID != "abc123" || l.Title != "Morning standup" || !l.IsStarred {
		t.Errorf("lifelog fields wrong: %+v", l)
	}
	if l.UpdatedAt != "2026-07-06T15:00:00.000Z" {
		t.Errorf("UpdatedAt = %q", l.UpdatedAt)
	}
	if len(l.Contents) != 1 || l.Contents[0].Type != "heading1" {
		t.Fatalf("contents wrong: %+v", l.Contents)
	}
	child := l.Contents[0].Children[0]
	if child.SpeakerName != "Ava" || child.SpeakerIdentifier != "user" {
		t.Errorf("child speaker wrong: %+v", child)
	}
	// null speakerIdentifier must decode to empty string, not error.
	if l.Contents[0].SpeakerIdentifier != "" {
		t.Errorf("null speakerIdentifier should be empty, got %q",
			l.Contents[0].SpeakerIdentifier)
	}
	if resp.Meta.Lifelogs.NextCursor != "cur_2" {
		t.Errorf("nextCursor = %q", resp.Meta.Lifelogs.NextCursor)
	}
}
