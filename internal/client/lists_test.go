package client

import (
	"context"
	"testing"

	"github.com/mudrii/gobird/internal/testutil"
	"github.com/mudrii/gobird/internal/types"
)

const listTimelineJSON = `{
	"data": {
		"list": {
			"tweets_timeline": {
				"timeline": {
					"instructions": [
						{
							"type": "TimelineAddEntries",
							"entries": [
								{
									"entryId": "tweet-555",
									"sortIndex": "2",
									"content": {
										"entryType": "TimelineTimelineItem",
										"itemContent": {
											"__typename": "TimelineTweet",
											"tweet_results": {
												"result": {
													"__typename": "Tweet",
													"rest_id": "555",
													"legacy": {
														"full_text": "List tweet",
														"created_at": "",
														"conversation_id_str": "555",
														"reply_count": 0,
														"retweet_count": 0,
														"favorite_count": 0,
														"user_id_str": "10"
													}
												}
											}
										}
									}
								},
								{
									"entryId": "cursor-bottom",
									"sortIndex": "0",
									"content": {
										"entryType": "TimelineTimelineCursor",
										"cursorType": "Bottom",
										"value": ""
									}
								}
							]
						}
					]
				}
			}
		}
	}
}`

func TestGetListTimeline_Basic(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, listTimelineJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{
		"ListLatestTweetsTimeline": "2TemLyqrMpTeAmysdbnVqw",
	})
	result, err := c.GetListTimeline(context.Background(), "list-123", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success")
	}
	if len(result.Items) != 1 {
		t.Fatalf("want 1 tweet, got %d", len(result.Items))
	}
	if result.Items[0].ID != "555" {
		t.Errorf("want tweet ID=555, got %q", result.Items[0].ID)
	}
}

func TestGetListTimeline_NoCursor(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, listTimelineJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{
		"ListLatestTweetsTimeline": "2TemLyqrMpTeAmysdbnVqw",
	})
	result, err := c.GetListTimeline(context.Background(), "list-999", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NextCursor != "" {
		t.Errorf("want empty NextCursor, got %q", result.NextCursor)
	}
}

func TestParseListTimelineResponse(t *testing.T) {
	page, err := parseListTimelineResponse([]byte(listTimelineJSON), 1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !page.Success {
		t.Fatal("expected success")
	}
	if len(page.Items) != 1 {
		t.Fatalf("want 1 tweet, got %d", len(page.Items))
	}
}

func TestMapWireList_Public(t *testing.T) {
	wl := &types.WireList{
		IDStr:       "list-42",
		Name:        "My List",
		Description: "A description",
		MemberCount: 10,
		Mode:        "Public",
	}
	tl := mapWireList(wl)
	if tl.ID != "list-42" {
		t.Errorf("want ID=list-42, got %q", tl.ID)
	}
	if tl.Name != "My List" {
		t.Errorf("want Name='My List', got %q", tl.Name)
	}
	if tl.IsPrivate {
		t.Error("want IsPrivate=false for Public list")
	}
}

func TestMapWireList_Private(t *testing.T) {
	wl := &types.WireList{
		IDStr: "list-99",
		Mode:  "Private",
	}
	tl := mapWireList(wl)
	if !tl.IsPrivate {
		t.Error("want IsPrivate=true for Private list")
	}
}

func TestGetListTimeline_Error(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(500, `{"errors":["server error"]}`))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{
		"ListLatestTweetsTimeline": "2TemLyqrMpTeAmysdbnVqw",
	})
	result, err := c.GetListTimeline(context.Background(), "list-err", nil)
	if err != nil {
		t.Log("error propagated correctly:", err)
		return
	}
	if result.Success {
		t.Fatal("expected failure")
	}
}
