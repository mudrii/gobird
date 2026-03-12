package client

import (
	"context"
	"testing"

	"github.com/mudrii/gobird/internal/testutil"
	"github.com/mudrii/gobird/internal/types"
)

const genericTimelineTweetsJSON = `{
	"data": {
		"timeline": {
			"timeline": {
				"instructions": [
					{
						"type": "TimelineAddEntries",
						"entries": [
							{
								"entryId": "tweet-news-1",
								"sortIndex": "1",
								"content": {
									"entryType": "TimelineTimelineItem",
									"itemContent": {
										"__typename": "TimelineTweet",
										"tweet_results": {
											"result": {
												"__typename": "Tweet",
												"rest_id": "news-tweet-1",
												"legacy": {
													"full_text": "Breaking news",
													"created_at": "",
													"conversation_id_str": "news-tweet-1",
													"reply_count": 0,
													"retweet_count": 0,
													"favorite_count": 0,
													"user_id_str": "1"
												}
											}
										}
									}
								}
							}
						]
					}
				]
			}
		}
	}
}`

func TestGetNews_DefaultTabs(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, genericTimelineTweetsJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{
		"GenericTimelineById": "uGSr7alSjR9v6QJAIaqSKQ",
	})
	items, err := c.GetNews(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 4 default tabs × 1 tweet each = at most 4 (deduped by ID → 1 since all return same).
	if len(items) == 0 {
		t.Fatal("expected at least one news item")
	}
}

func TestGetNews_SpecificTab(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, genericTimelineTweetsJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{
		"GenericTimelineById": "uGSr7alSjR9v6QJAIaqSKQ",
	})
	items, err := c.GetNews(context.Background(), &types.NewsOptions{
		Tabs:     []string{"news"},
		MaxCount: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one news item")
	}
}

func TestGetNews_UnknownTab(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, genericTimelineTweetsJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{
		"GenericTimelineById": "uGSr7alSjR9v6QJAIaqSKQ",
	})
	items, err := c.GetNews(context.Background(), &types.NewsOptions{
		Tabs: []string{"nonexistent_tab"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("want 0 items for unknown tab, got %d", len(items))
	}
}

func TestMapNewsItem(t *testing.T) {
	postCount := 42
	w := &newsItemWire{
		RestID:      "trend-1",
		Name:        "Breaking News",
		Description: "A headline",
		Category:    "news",
		TimeAgo:     "2h",
		IsAINews:    false,
		PostCount:   &postCount,
		TrendURL:    "https://x.com/search?q=news",
	}
	item := mapNewsItem(w, "entry-1")
	if item.ID != "trend-1" {
		t.Errorf("want ID=trend-1, got %q", item.ID)
	}
	if item.Headline != "Breaking News" {
		t.Errorf("want Headline='Breaking News', got %q", item.Headline)
	}
	if item.Category != "news" {
		t.Errorf("want Category=news, got %q", item.Category)
	}
	if item.PostCount == nil || *item.PostCount != 42 {
		t.Errorf("want PostCount=42, got %v", item.PostCount)
	}
}

func TestMapNewsItem_FallbackID(t *testing.T) {
	w := &newsItemWire{
		RestID: "",
		Name:   "Fallback",
	}
	item := mapNewsItem(w, "entry-fallback")
	if item.ID != "entry-fallback" {
		t.Errorf("want ID=entry-fallback when rest_id empty, got %q", item.ID)
	}
}

func TestParseGenericTimelineResponse(t *testing.T) {
	items, err := parseGenericTimelineResponse([]byte(genericTimelineTweetsJSON), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one item")
	}
}

func TestDefaultNewsTabs_Count(t *testing.T) {
	if len(DefaultNewsTabs) != 4 {
		t.Errorf("want 4 default news tabs, got %d", len(DefaultNewsTabs))
	}
}

func TestDefaultNewsTabs_NoTrending(t *testing.T) {
	for _, tab := range DefaultNewsTabs {
		if tab == "trending" {
			t.Error("DefaultNewsTabs must not include 'trending'")
		}
	}
}
