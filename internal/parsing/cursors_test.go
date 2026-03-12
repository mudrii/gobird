package parsing_test

import (
	"testing"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

func TestExtractCursorFromInstructions(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				{Content: types.WireContent{CursorType: "Top", Value: "top-cursor"}},
				{Content: types.WireContent{CursorType: "Bottom", Value: "bottom-cursor"}},
			},
		},
	}
	got := parsing.ExtractCursorFromInstructions(instructions)
	if got != "bottom-cursor" {
		t.Errorf("want bottom-cursor, got %q", got)
	}
}

func TestExtractCursorNotEntryType(t *testing.T) {
	// Correction #7: cursorType is checked, NOT entryType.
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				{Content: types.WireContent{EntryType: "TimelineCursor", CursorType: "", Value: "should-not-match"}},
				{Content: types.WireContent{CursorType: "Bottom", Value: "correct"}},
			},
		},
	}
	got := parsing.ExtractCursorFromInstructions(instructions)
	if got != "correct" {
		t.Errorf("want correct, got %q", got)
	}
}
