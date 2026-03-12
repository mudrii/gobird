package parsing

import (
	"github.com/mudrii/gobird/internal/types"
)

// MapList converts a raw wire list to a normalized TwitterList.
func MapList(raw *types.WireList) *types.TwitterList {
	if raw == nil {
		return nil
	}
	l := &types.TwitterList{
		ID:              raw.IDStr,
		Name:            raw.Name,
		Description:     raw.Description,
		MemberCount:     raw.MemberCount,
		SubscriberCount: raw.SubscriberCount,
		IsPrivate:       raw.Mode == "private",
		CreatedAt:       raw.CreatedAt,
	}
	if raw.UserResults.Result != nil && raw.UserResults.Result.Legacy != nil {
		ul := raw.UserResults.Result.Legacy
		l.Owner = &types.ListOwner{
			ID:       raw.UserResults.Result.RestID,
			Username: ul.ScreenName,
			Name:     ul.Name,
		}
	}
	return l
}

// ParseListsFromInstructions collects normalized TwitterList items from timeline instructions.
func ParseListsFromInstructions(instructions []types.WireTimelineInstruction) []types.TwitterList {
	// Lists appear in a different response shape — handled directly by client/lists.go.
	// This function is a placeholder for completeness.
	_ = instructions
	return nil
}
