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
		IsPrivate:       raw.Mode == "Private",
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

