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

// ParseUsersFromInstructions collects normalized TwitterUser items from
// timeline instructions. Handles UserWithVisibilityResults unwrapping.
func ParseUsersFromInstructions(instructions []types.WireTimelineInstruction) []types.TwitterUser {
	var users []types.TwitterUser
	seen := map[string]bool{}
	for _, inst := range instructions {
		for _, entry := range inst.Entries {
			if entry.Content.ItemContent != nil {
				if ur := entry.Content.ItemContent.UserResult; ur != nil && ur.Result != nil {
					u := MapUser(ur.Result)
					if u != nil && !seen[u.ID] {
						seen[u.ID] = true
						users = append(users, *u)
					}
				}
			}
		}
	}
	return users
}
