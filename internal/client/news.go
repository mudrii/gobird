package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

// GetNews fetches news items from the specified tabs (or DefaultNewsTabs if none specified).
// Uses GenericTimelineById (corrections #34, #67).
func (c *Client) GetNews(ctx context.Context, opts *types.NewsOptions) ([]types.NewsItem, error) {
	if opts == nil {
		opts = &types.NewsOptions{}
	}
	tabs := opts.Tabs
	if len(tabs) == 0 {
		tabs = DefaultNewsTabs
	}
	maxCount := opts.MaxCount
	if maxCount <= 0 {
		maxCount = 20
	}

	var allItems []types.NewsItem
	seen := map[string]bool{}

	for _, tab := range tabs {
		timelineID, ok := GenericTimelineTabIDs[tab]
		if !ok {
			continue
		}
		items, err := c.fetchGenericTimeline(ctx, timelineID, maxCount)
		if err != nil {
			continue
		}
		for _, item := range items {
			if !seen[item.ID] {
				seen[item.ID] = true
				if opts.IncludeRaw {
					// IncludeRaw is set on item if applicable; leave as-is since items come from parsing.
				}
				allItems = append(allItems, item)
			}
		}
	}

	return allItems, nil
}

// fetchGenericTimeline fetches items from GenericTimelineById for a single tab.
// Correction #34, #67: variables: {"timelineId":"<tab-timeline-id>","count":"<maxCount*2>","includePromotedContent":false}
// Response path: data.timeline.timeline.instructions.
// Uses buildExploreFeatures().
func (c *Client) fetchGenericTimeline(ctx context.Context, timelineID string, maxCount int) ([]types.NewsItem, error) {
	queryID := c.getQueryID("GenericTimelineById")
	features := buildExploreFeatures()

	vars := map[string]any{
		"timelineId":             timelineID,
		"count":                  strconv.Itoa(maxCount * 2),
		"includePromotedContent": false,
	}

	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return nil, err
	}
	featuresJSON, err := json.Marshal(features)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s/%s/GenericTimelineById?variables=%s&features=%s",
		GraphQLBaseURL, queryID,
		url.QueryEscape(string(varsJSON)),
		url.QueryEscape(string(featuresJSON)),
	)

	body, err := c.doGET(ctx, reqURL, c.getJsonHeaders())
	if err != nil {
		return nil, err
	}

	return parseGenericTimelineResponse(body)
}

// parseGenericTimelineResponse parses the GenericTimelineById response.
// Response path: data.timeline.timeline.instructions.
func parseGenericTimelineResponse(body []byte) ([]types.NewsItem, error) {
	var env struct {
		Data struct {
			Timeline struct {
				Timeline struct {
					Instructions []types.WireTimelineInstruction `json:"instructions"`
				} `json:"timeline"`
			} `json:"timeline"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	instructions := env.Data.Timeline.Timeline.Instructions
	return parseNewsItemsFromInstructions(instructions, body)
}

// parseNewsItemsFromInstructions extracts NewsItem entries from timeline instructions.
// News items are trend/explore entries with a different wire shape than tweets.
func parseNewsItemsFromInstructions(instructions []types.WireTimelineInstruction, body []byte) ([]types.NewsItem, error) {
	// First, try to extract tweet-based news items.
	tweets := parsing.ParseTweetsFromInstructions(instructions)
	if len(tweets) > 0 {
		var items []types.NewsItem
		for _, t := range tweets {
			item := types.NewsItem{
				ID:       t.ID,
				Headline: t.Text,
			}
			items = append(items, item)
		}
		return items, nil
	}

	// Fall back to raw JSON parsing for explore/trend entries.
	return parseNewsItemsRaw(body)
}

// parseNewsItemsRaw parses news items from the raw GenericTimelineById response.
// Trend/explore entries have a different structure than tweet entries.
func parseNewsItemsRaw(body []byte) ([]types.NewsItem, error) {
	var env struct {
		Data struct {
			Timeline struct {
				Timeline struct {
					Instructions []struct {
						Entries []struct {
							EntryID string `json:"entryId"`
							Content struct {
								ItemContent *struct {
									TrendResult *struct {
										Result *newsItemWire `json:"result"`
									} `json:"trend_results"`
									TypeName string `json:"__typename"`
								} `json:"itemContent"`
								Items []struct {
									Item struct {
										ItemContent *struct {
											TrendResult *struct {
												Result *newsItemWire `json:"result"`
											} `json:"trend_results"`
										} `json:"itemContent"`
									} `json:"item"`
								} `json:"items"`
							} `json:"content"`
						} `json:"entries"`
					} `json:"instructions"`
				} `json:"timeline"`
			} `json:"timeline"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}

	var items []types.NewsItem
	seen := map[string]bool{}

	for _, inst := range env.Data.Timeline.Timeline.Instructions {
		for _, entry := range inst.Entries {
			if entry.Content.ItemContent != nil {
				if tr := entry.Content.ItemContent.TrendResult; tr != nil && tr.Result != nil {
					item := mapNewsItem(tr.Result, entry.EntryID)
					if !seen[item.ID] {
						seen[item.ID] = true
						items = append(items, item)
					}
				}
			}
			for _, moduleItem := range entry.Content.Items {
				if ic := moduleItem.Item.ItemContent; ic != nil {
					if tr := ic.TrendResult; tr != nil && tr.Result != nil {
						item := mapNewsItem(tr.Result, "")
						if !seen[item.ID] {
							seen[item.ID] = true
							items = append(items, item)
						}
					}
				}
			}
		}
	}

	return items, nil
}

// newsItemWire is the wire shape for a trend/explore news item.
type newsItemWire struct {
	RestID      string `json:"rest_id"`
	Name        string `json:"name"`
	TrendURL    string `json:"trend_url"`
	Description string `json:"description"`
	Category    string `json:"category"`
	TimeAgo     string `json:"time_ago"`
	IsAINews    bool   `json:"is_ai_news"`
	PostCount   *int   `json:"post_count"`
}

// mapNewsItem converts a newsItemWire to a NewsItem.
func mapNewsItem(w *newsItemWire, entryID string) types.NewsItem {
	id := w.RestID
	if id == "" {
		id = entryID
	}
	return types.NewsItem{
		ID:          id,
		Headline:    w.Name,
		Category:    w.Category,
		TimeAgo:     w.TimeAgo,
		PostCount:   w.PostCount,
		Description: w.Description,
		URL:         w.TrendURL,
		IsAiNews:    w.IsAINews,
	}
}
