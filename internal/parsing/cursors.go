// Package parsing implements all GraphQL response parsing algorithms.
package parsing

import (
	"github.com/mudrii/gobird/internal/types"
)

// ExtractCursorFromInstructions scans timeline instructions for a Bottom cursor.
// Correction #7: checks entry.content.cursorType ONLY — not entryType, not module items.
func ExtractCursorFromInstructions(instructions []types.WireTimelineInstruction) string {
	for _, inst := range instructions {
		for _, entry := range inst.Entries {
			if entry.Content.CursorType == "Bottom" {
				return entry.Content.Value
			}
		}
		if inst.Entry != nil && inst.Entry.Content.CursorType == "Bottom" {
			return inst.Entry.Content.Value
		}
	}
	return ""
}
