package parsing

import (
	"github.com/mudrii/gobird/internal/types"
)

// UnwrapUserResult unwraps a UserWithVisibilityResults wrapper if present.
func UnwrapUserResult(raw *types.WireRawUser) *types.WireRawUser {
	if raw != nil && raw.TypeName == "UserWithVisibilityResults" && raw.User != nil {
		return raw.User
	}
	return raw
}

// MapUser converts a wire user result to a normalized TwitterUser.
// IsBlueVerified is taken from the top-level field, not from legacy (correction #8).
func MapUser(raw *types.WireRawUser) *types.TwitterUser {
	raw = UnwrapUserResult(raw)
	if raw == nil || raw.TypeName == "UserUnavailable" {
		return nil
	}
	u := &types.TwitterUser{
		ID:             raw.RestID,
		IsBlueVerified: raw.IsBlueVerified,
	}
	if raw.Legacy != nil {
		u.Username = raw.Legacy.ScreenName
		u.Name = raw.Legacy.Name
		u.Description = raw.Legacy.Description
		u.FollowersCount = raw.Legacy.FollowersCount
		u.FollowingCount = raw.Legacy.FriendsCount
		u.ProfileImageURL = raw.Legacy.ProfileImageURLHTTPS
		u.CreatedAt = raw.Legacy.CreatedAt
	}
	return u
}

// collectUserFromItemContent extracts a user from a WireItemContent if present.
func collectUserFromItemContent(ic *types.WireItemContent, seen map[string]bool, users *[]types.TwitterUser) {
	if ic == nil {
		return
	}
	ur := ic.UserResult
	if ur == nil || ur.Result == nil {
		return
	}
	u := MapUser(ur.Result)
	if u != nil && !seen[u.ID] {
		seen[u.ID] = true
		*users = append(*users, *u)
	}
}

// ParseUsersFromInstructions collects normalized TwitterUser items from
// timeline instructions. Handles UserWithVisibilityResults unwrapping.
// Checks both inst.Entries (multi-entry) and inst.Entry (single-entry) forms,
// and also scans module items within each entry.
func ParseUsersFromInstructions(instructions []types.WireTimelineInstruction) []types.TwitterUser {
	var users []types.TwitterUser
	seen := map[string]bool{}
	for _, inst := range instructions {
		for _, entry := range inst.Entries {
			collectUserFromItemContent(entry.Content.ItemContent, seen, &users)
			// Also scan module items within each entry.
			for _, item := range entry.Content.Items {
				collectUserFromItemContent(item.Item.ItemContent, seen, &users)
			}
		}
		// Some instructions use the singular Entry field instead of Entries.
		if inst.Entry != nil {
			collectUserFromItemContent(inst.Entry.Content.ItemContent, seen, &users)
			for _, item := range inst.Entry.Content.Items {
				collectUserFromItemContent(item.Item.ItemContent, seen, &users)
			}
		}
	}
	return users
}
