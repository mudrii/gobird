package client

import (
	"context"
	"net/http"
	"testing"

	"github.com/mudrii/gobird/internal/testutil"
	"github.com/mudrii/gobird/internal/types"
)

const userTweetsJSON = `{
	"data": {
		"user": {
			"result": {
				"timeline": {
					"timeline": {
						"instructions": [
							{
								"type": "TimelineAddEntries",
								"entries": [
									{
										"entryId": "tweet-1",
										"sortIndex": "1",
										"content": {
											"entryType": "TimelineTimelineItem",
											"itemContent": {
												"__typename": "TimelineTweet",
												"tweet_results": {
													"result": {
														"__typename": "Tweet",
														"rest_id": "1001",
														"is_blue_verified": false,
														"core": {
															"user_results": {
																"result": {
																	"__typename": "User",
																	"rest_id": "42",
																	"legacy": {
																		"screen_name": "tweeter",
																		"name": "Tweeter"
																	}
																}
															}
														},
														"legacy": {
															"full_text": "Hello world",
															"created_at": "Mon Jan 01 00:00:00 +0000 2024",
															"conversation_id_str": "1001",
															"reply_count": 0,
															"retweet_count": 5,
															"favorite_count": 10,
															"user_id_str": "42"
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
											"value": "next-page-cursor"
										}
									}
								]
							}
						]
					}
				}
			}
		}
	}
}`

func TestGetUserTweetsPaged(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, userTweetsJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{"UserTweets": "Wms1GvIiHXAPBaCr9KblaA"})
	page, err := c.GetUserTweetsPaged(context.Background(), "42", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !page.Success {
		t.Fatal("expected success")
	}
	if len(page.Items) != 1 {
		t.Fatalf("want 1 tweet, got %d", len(page.Items))
	}
	if page.Items[0].ID != "1001" {
		t.Errorf("want tweet ID=1001, got %q", page.Items[0].ID)
	}
	if page.NextCursor != "next-page-cursor" {
		t.Errorf("want cursor=next-page-cursor, got %q", page.NextCursor)
	}
}

func TestGetUserTweets_SinglePage(t *testing.T) {
	const noNextJSON = `{
		"data": {
			"user": {
				"result": {
					"timeline": {
						"timeline": {
							"instructions": [
								{
									"type": "TimelineAddEntries",
									"entries": [
										{
											"entryId": "tweet-2",
											"sortIndex": "1",
											"content": {
												"entryType": "TimelineTimelineItem",
												"itemContent": {
													"__typename": "TimelineTweet",
													"tweet_results": {
														"result": {
															"__typename": "Tweet",
															"rest_id": "2002",
															"legacy": {
																"full_text": "No cursor",
																"created_at": "",
																"conversation_id_str": "2002",
																"reply_count": 0,
																"retweet_count": 0,
																"favorite_count": 0,
																"user_id_str": "99"
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
			}
		}
	}`

	srv := testutil.NewTestServer(testutil.StaticHandler(200, noNextJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{"UserTweets": "Wms1GvIiHXAPBaCr9KblaA"})
	result, err := c.GetUserTweets(context.Background(), "99", &types.UserTweetsOptions{
		FetchOptions: types.FetchOptions{Limit: 20},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success")
	}
	if len(result.Items) != 1 {
		t.Fatalf("want 1 tweet, got %d", len(result.Items))
	}
	if result.NextCursor != "" {
		t.Errorf("want empty cursor, got %q", result.NextCursor)
	}
}

func TestGetUserTweets_HardMaxPages(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, userTweetsJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{"UserTweets": "Wms1GvIiHXAPBaCr9KblaA"})

	// limit=200 → calcPages=10; maxPages=15 → capped to 10.
	// Mock returns cursor "next-page-cursor" always; on page 2 it equals the
	// cursor we send, so pagination stops with success.
	result, err := c.GetUserTweets(context.Background(), "42", &types.UserTweetsOptions{
		FetchOptions: types.FetchOptions{Limit: 200, MaxPages: 15},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = result
}

func TestParseUserTweetsResponse(t *testing.T) {
	page, err := parseUserTweetsResponse([]byte(userTweetsJSON), 1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("want 1 tweet, got %d", len(page.Items))
	}
	if page.Items[0].Text != "Hello world" {
		t.Errorf("want text='Hello world', got %q", page.Items[0].Text)
	}
	if page.NextCursor != "next-page-cursor" {
		t.Errorf("want cursor='next-page-cursor', got %q", page.NextCursor)
	}
}

func newTestClientWith(baseURL string, queryCache map[string]string) *Client {
	transport := testutil.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
		testReq, _ := http.NewRequestWithContext(r.Context(), r.Method, baseURL+r.URL.RequestURI(), r.Body)
		for k, vs := range r.Header {
			testReq.Header[k] = vs
		}
		return http.DefaultTransport.RoundTrip(testReq)
	})
	c := New("fake-auth", "fake-ct0", &Options{
		HTTPClient:   &http.Client{Transport: transport},
		QueryIDCache: queryCache,
	})
	c.scraper = func(_ context.Context) map[string]string { return nil }
	return c
}
