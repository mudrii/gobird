package client

import (
	"context"
	"net/http"
	"testing"

	"github.com/mudrii/gobird/internal/testutil"
	"github.com/mudrii/gobird/internal/types"
)

const tweetDetailJSON = `{
	"data": {
		"tweetResult": {
			"result": {
				"__typename": "Tweet",
				"rest_id": "9999",
				"is_blue_verified": true,
				"core": {
					"user_results": {
						"result": {
							"__typename": "User",
							"rest_id": "111",
							"legacy": {
								"screen_name": "author",
								"name": "The Author"
							}
						}
					}
				},
				"legacy": {
					"full_text": "A detailed tweet",
					"created_at": "Mon Jan 01 00:00:00 +0000 2024",
					"conversation_id_str": "9999",
					"reply_count": 3,
					"retweet_count": 7,
					"favorite_count": 42,
					"user_id_str": "111"
				}
			}
		}
	}
}`

const threadedConversationJSON = `{
	"data": {
		"threaded_conversation_with_injections_v2": {
			"instructions": [
				{
					"type": "TimelineAddEntries",
					"entries": [
						{
							"entryId": "tweet-9999",
							"sortIndex": "2",
							"content": {
								"entryType": "TimelineTimelineItem",
								"itemContent": {
									"__typename": "TimelineTweet",
									"tweet_results": {
										"result": {
											"__typename": "Tweet",
											"rest_id": "9999",
											"legacy": {
												"full_text": "Root tweet",
												"created_at": "",
												"conversation_id_str": "9999",
												"reply_count": 1,
												"retweet_count": 0,
												"favorite_count": 0,
												"user_id_str": "111"
											}
										}
									}
								}
							}
						},
						{
							"entryId": "tweet-10000",
							"sortIndex": "1",
							"content": {
								"entryType": "TimelineTimelineItem",
								"itemContent": {
									"__typename": "TimelineTweet",
									"tweet_results": {
										"result": {
											"__typename": "Tweet",
											"rest_id": "10000",
											"legacy": {
												"full_text": "Reply tweet",
												"created_at": "",
												"conversation_id_str": "9999",
												"in_reply_to_status_id_str": "9999",
												"reply_count": 0,
												"retweet_count": 0,
												"favorite_count": 0,
												"user_id_str": "111"
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
}`

func TestGetTweet(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, tweetDetailJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{"TweetDetail": "_NvJCnIjOW__EP5-RF197A"})
	tweet, err := c.GetTweet(context.Background(), "9999", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tweet.ID != "9999" {
		t.Errorf("want ID=9999, got %q", tweet.ID)
	}
	if tweet.Text != "A detailed tweet" {
		t.Errorf("want text='A detailed tweet', got %q", tweet.Text)
	}
	if !tweet.IsBlueVerified {
		t.Error("want IsBlueVerified=true")
	}
}

func TestGetTweet_IncludeRaw(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, tweetDetailJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{"TweetDetail": "_NvJCnIjOW__EP5-RF197A"})
	tweet, err := c.GetTweet(context.Background(), "9999", &types.TweetDetailOptions{IncludeRaw: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tweet.Raw == nil {
		t.Error("want non-nil Raw when IncludeRaw=true")
	}
}

func TestGetTweet_NotFound(t *testing.T) {
	const emptyJSON = `{"data":{"tweetResult":null}}`
	srv := testutil.NewTestServer(testutil.StaticHandler(200, emptyJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{"TweetDetail": "_NvJCnIjOW__EP5-RF197A"})
	_, err := c.GetTweet(context.Background(), "0000", nil)
	if err == nil {
		t.Fatal("expected error for missing tweetResult")
	}
}

func TestGetReplies(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, threadedConversationJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{"TweetDetail": "_NvJCnIjOW__EP5-RF197A"})
	result, err := c.GetReplies(context.Background(), "9999", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success")
	}
	if len(result.Items) < 1 {
		t.Fatalf("want at least 1 item, got %d", len(result.Items))
	}
}

func TestGetThread_AuthorChain(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, threadedConversationJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{"TweetDetail": "_NvJCnIjOW__EP5-RF197A"})
	tweets, err := c.GetThread(context.Background(), "9999", &types.ThreadOptions{
		FilterMode: "author_chain",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Both tweets have authorID=111, so all should pass filter.
	if len(tweets) == 0 {
		t.Fatal("expected at least one tweet in thread")
	}
}

func TestGetThread_FullChain(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, threadedConversationJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{"TweetDetail": "_NvJCnIjOW__EP5-RF197A"})
	tweets, err := c.GetThread(context.Background(), "9999", &types.ThreadOptions{
		FilterMode: "full_chain",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tweets) < 2 {
		t.Fatalf("want 2 tweets in full chain, got %d", len(tweets))
	}
}

func TestGetTweet_GET404ThenPOST(t *testing.T) {
	// First request: GET → 404; second request: POST → 200 with valid JSON.
	callCount := 0
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"errors":[{"message":"Not Found"}]}`))
			return
		}
		// POST → success.
		w.WriteHeader(200)
		_, _ = w.Write([]byte(tweetDetailJSON))
	}))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{"TweetDetail": "_NvJCnIjOW__EP5-RF197A"})
	tweet, err := c.GetTweet(context.Background(), "9999", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tweet.ID != "9999" {
		t.Errorf("want ID=9999, got %q", tweet.ID)
	}
}

func TestBuildTweetDetailVars_NoCursor(t *testing.T) {
	vars := buildTweetDetailVars("123", "")
	if _, ok := vars["cursor"]; ok {
		t.Error("cursor should not be set when empty")
	}
	if vars["focalTweetId"] != "123" {
		t.Errorf("want focalTweetId=123, got %v", vars["focalTweetId"])
	}
}

func TestBuildTweetDetailVars_WithCursor(t *testing.T) {
	vars := buildTweetDetailVars("123", "abc")
	if vars["cursor"] != "abc" {
		t.Errorf("want cursor=abc, got %v", vars["cursor"])
	}
}
