package catalog

import (
	"strings"
	"testing"
	"time"

	"github.com/CLarkin-Arcalea/limitless-catalog/internal/limitless"
)

// corruptedLog mirrors a real-world record: epoch-zero startTime, sane
// content-node timestamps, sane endTime.
func corruptedLog() limitless.Lifelog {
	return limitless.Lifelog{
		ID: "corrupt1", Title: "Recovered meeting",
		StartTime: "1970-01-01T00:00:00Z",
		EndTime:   "2025-09-25T21:14:59Z",
		UpdatedAt: "2025-09-25T22:00:00Z",
		Contents: []limitless.ContentNode{
			{Type: "heading1", Content: "Recovered meeting", Children: []limitless.ContentNode{
				{Type: "blockquote", Content: "recording in progress",
					SpeakerName: "Alex", StartTime: "2025-09-25T20:57:39+00:00"},
				{Type: "blockquote", Content: "let us begin",
					SpeakerName: "Alex", StartTime: "2025-09-25T20:58:02Z"},
			}},
		},
	}
}

func TestBuildRepairsEpochStartFromContentNodes(t *testing.T) {
	chicago, _ := time.LoadLocation("America/Chicago")
	r, err := Build(corruptedLog(), chicago)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if r.StartUTC != "2025-09-25T20:57:39Z" {
		t.Errorf("StartUTC = %q, want repaired 2025-09-25T20:57:39Z", r.StartUTC)
	}
	if r.LocalDate != "2025-09-25" {
		t.Errorf("LocalDate = %q, want 2025-09-25 (Chicago)", r.LocalDate)
	}
	if r.DurationMin < 17.3 || r.DurationMin > 17.4 {
		t.Errorf("DurationMin = %v, want ~17.33 (20:57:39 to 21:14:59)", r.DurationMin)
	}
	if !strings.Contains(r.RawJSON, "1970-01-01T00:00:00Z") {
		t.Error("RawJSON must preserve the original corrupted startTime")
	}
}

func TestBuildRepairsFallsBackToEndTime(t *testing.T) {
	l := corruptedLog()
	// Strip all content-node timestamps: only EndTime is usable.
	l.Contents[0].Children[0].StartTime = ""
	l.Contents[0].Children[1].StartTime = ""
	r, err := Build(l, time.UTC)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if r.StartUTC != "2025-09-25T21:14:59Z" {
		t.Errorf("StartUTC = %q, want EndTime fallback", r.StartUTC)
	}
	if r.DurationMin != 0 {
		t.Errorf("DurationMin = %v, want 0 for point-in-time fallback", r.DurationMin)
	}
}

func TestBuildUnrecoverableStillErrors(t *testing.T) {
	l := corruptedLog()
	l.EndTime = "garbage"
	l.Contents[0].Children[0].StartTime = ""
	l.Contents[0].Children[1].StartTime = "also-garbage"
	if _, err := Build(l, time.UTC); err == nil {
		t.Fatal("want error when no usable timestamp exists anywhere")
	}
}

func TestBuildRepairsPreFloorEndTime(t *testing.T) {
	l := corruptedLog()
	l.StartTime = "2025-09-25T20:57:39Z" // sane start
	l.EndTime = "1970-01-01T00:00:00Z"   // corrupted end
	r, err := Build(l, time.UTC)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if r.EndUTC != "2025-09-25T20:57:39Z" {
		t.Errorf("EndUTC = %q, want point-in-time fallback to start", r.EndUTC)
	}
	if r.DurationMin != 0 {
		t.Errorf("DurationMin = %v, want 0", r.DurationMin)
	}
}

func TestBuildPreFloorContentNodeIgnored(t *testing.T) {
	l := corruptedLog()
	// A content node that is itself pre-floor garbage must be skipped in
	// favor of the next sane node.
	l.Contents[0].Children[0].StartTime = "1970-01-01T00:00:05Z"
	r, err := Build(l, time.UTC)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if r.StartUTC != "2025-09-25T20:58:02Z" {
		t.Errorf("StartUTC = %q, want the first SANE node time", r.StartUTC)
	}
}
