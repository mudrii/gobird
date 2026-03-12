package parsing

import (
	"github.com/mudrii/gobird/internal/types"
)

// ParseNewsItemFromContent attempts to extract a NewsItem from a raw content map.
// Used by client/news.go when parsing GenericTimelineById responses.
func ParseNewsItemFromContent(content map[string]any) *types.NewsItem {
	if content == nil {
		return nil
	}
	item := &types.NewsItem{}

	if id, ok := content["id"].(string); ok {
		item.ID = id
	}
	if headline, ok := content["name"].(string); ok {
		item.Headline = headline
	} else if headline, ok := content["headline"].(string); ok {
		item.Headline = headline
	}
	if cat, ok := content["category"].(string); ok {
		item.Category = cat
	}
	if timeAgo, ok := content["time_ago"].(string); ok {
		item.TimeAgo = timeAgo
	}
	if desc, ok := content["description"].(string); ok {
		item.Description = desc
	}
	if u, ok := content["url"].(string); ok {
		item.URL = u
	}
	if isAI, ok := content["is_ai_news"].(bool); ok {
		item.IsAiNews = isAI
	}
	if postCount, ok := content["tweet_count"].(float64); ok {
		n := int(postCount)
		item.PostCount = &n
	}
	return item
}
