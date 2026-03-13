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

func TestExtractCursorFromInstructions_Bottom(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				{Content: types.WireContent{CursorType: "Bottom", Value: "next-page-cursor"}},
			},
		},
	}
	got := parsing.ExtractCursorFromInstructions(instructions)
	if got != "next-page-cursor" {
		t.Errorf("want next-page-cursor, got %q", got)
	}
}

func TestExtractCursorFromInstructions_Top(t *testing.T) {
	// The current implementation only matches Bottom cursors.
	// A Top-only instruction set should return empty string.
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				{Content: types.WireContent{CursorType: "Top", Value: "top-only"}},
			},
		},
	}
	got := parsing.ExtractCursorFromInstructions(instructions)
	if got != "" {
		t.Errorf("Top-only cursor should not be returned by ExtractCursorFromInstructions, got %q", got)
	}
}

func TestExtractCursorFromInstructions_Empty(t *testing.T) {
	got := parsing.ExtractCursorFromInstructions(nil)
	if got != "" {
		t.Errorf("want empty for nil instructions, got %q", got)
	}
}

func TestExtractCursorFromInstructions_EmptyEntries(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{Entries: []types.WireEntry{}},
	}
	got := parsing.ExtractCursorFromInstructions(instructions)
	if got != "" {
		t.Errorf("want empty for empty entries, got %q", got)
	}
}

func TestExtractCursorFromInstructions_SingleEntry(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entry: &types.WireEntry{
				Content: types.WireContent{CursorType: "Bottom", Value: "single-entry-cursor"},
			},
		},
	}
	got := parsing.ExtractCursorFromInstructions(instructions)
	if got != "single-entry-cursor" {
		t.Errorf("want single-entry-cursor from inst.Entry, got %q", got)
	}
}

func TestExtractCursorFromInstructions_MultipleInstructions(t *testing.T) {
	instructions := []types.WireTimelineInstruction{
		{
			Entries: []types.WireEntry{
				{Content: types.WireContent{CursorType: "Top", Value: "top"}},
			},
		},
		{
			Entries: []types.WireEntry{
				{Content: types.WireContent{CursorType: "Bottom", Value: "bottom-second"}},
			},
		},
	}
	got := parsing.ExtractCursorFromInstructions(instructions)
	if got != "bottom-second" {
		t.Errorf("want bottom-second, got %q", got)
	}
}
