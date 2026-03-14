package bird_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mudrii/gobird/internal/testutil"
	"github.com/mudrii/gobird/pkg/bird"
)

// newMockClient creates a bird.Client whose HTTP calls are routed to srv.
// queryIDs is a map of operation → query ID to inject into the client cache.
func newMockClient(t *testing.T, srv *httptest.Server, queryIDs map[string]string) *bird.Client {
	t.Helper()
	httpClient := testutil.NewHTTPClientForServer(srv)
	opts := &bird.ClientOptions{
		HTTPClient:        httpClient,
		QueryIDCache:      queryIDs,
		RequestsPerSecond: -1,
	}
	c, err := bird.NewWithTokens("fake-auth", "fake-ct0", opts)
	if err != nil {
		t.Fatalf("newMockClient: %v", err)
	}
	return c
}

// tweetDetailResp returns a minimal tweetResult JSON response.
func tweetDetailResp(id, text string) string {
	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"tweetResult": map[string]any{
				"result": map[string]any{
					"__typename": "Tweet",
					"rest_id":    id,
					"legacy": map[string]any{
						"full_text":           text,
						"created_at":          "",
						"conversation_id_str": id,
						"reply_count":         0,
						"retweet_count":       0,
						"favorite_count":      0,
						"user_id_str":         "u1",
					},
				},
			},
		},
	})
	return string(body)
}

// homeTimelineBody returns a minimal HomeTimeline response with 1 tweet.
func homeTimelineBody(id string) string {
	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"home": map[string]any{
				"home_timeline_urt": map[string]any{
					"instructions": []any{
						map[string]any{
							"entries": []any{
								map[string]any{
									"entryId": "tweet-" + id,
									"content": map[string]any{
										"itemContent": map[string]any{
											"tweet_results": map[string]any{
												"result": map[string]any{
													"__typename": "Tweet",
													"rest_id":    id,
													"legacy": map[string]any{
														"full_text": "home tweet " + id,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	return string(body)
}

// bookmarksBody returns a minimal Bookmarks response with one tweet.
func bookmarksBody(id string) string {
	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"bookmark_timeline_v2": map[string]any{
				"timeline": map[string]any{
					"instructions": []any{
						map[string]any{
							"entries": []any{
								map[string]any{
									"entryId": "tweet-" + id,
									"content": map[string]any{
										"itemContent": map[string]any{
											"tweet_results": map[string]any{
												"result": map[string]any{
													"__typename": "Tweet",
													"rest_id":    id,
													"legacy":     map[string]any{"full_text": "bookmark " + id},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	return string(body)
}

func TestClient_GetTweet_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, tweetDetailResp("wrap1", "wrapped tweet")))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"TweetDetail": "testQID"})
	tweet, err := c.GetTweet(context.Background(), "wrap1", nil)
	if err != nil {
		t.Fatalf("GetTweet: %v", err)
	}
	if tweet.ID != "wrap1" {
		t.Errorf("ID: want wrap1, got %q", tweet.ID)
	}
	if tweet.Text != "wrapped tweet" {
		t.Errorf("Text: want %q, got %q", "wrapped tweet", tweet.Text)
	}
}

func TestClient_GetHomeTimeline_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, homeTimelineBody("home1")))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"HomeTimeline": "homeQID"})
	result := c.GetHomeTimeline(context.Background(), nil)
	if !result.Success {
		t.Fatalf("GetHomeTimeline: %v", result.Error)
	}
	if len(result.Items) != 1 {
		t.Errorf("want 1 item, got %d", len(result.Items))
	}
}

func TestClient_GetHomeLatestTimeline_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, homeTimelineBody("latest1")))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"HomeLatestTimeline": "latestQID"})
	result := c.GetHomeLatestTimeline(context.Background(), nil)
	if !result.Success {
		t.Fatalf("GetHomeLatestTimeline: %v", result.Error)
	}
	if len(result.Items) != 1 {
		t.Errorf("want 1 item, got %d", len(result.Items))
	}
}

func TestClient_GetBookmarks_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, bookmarksBody("bk1")))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"Bookmarks": "booksQID"})
	result := c.GetBookmarks(context.Background(), nil)
	if !result.Success {
		t.Fatalf("GetBookmarks: %v", result.Error)
	}
	if len(result.Items) != 1 {
		t.Errorf("want 1 bookmark, got %d", len(result.Items))
	}
}

func TestClient_Search_Wrapper(t *testing.T) {
	searchBody, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"search_by_raw_query": map[string]any{
				"search_timeline": map[string]any{
					"timeline": map[string]any{
						"instructions": []any{
							map[string]any{
								"entries": []any{
									map[string]any{
										"entryId": "tweet-s1",
										"content": map[string]any{
											"itemContent": map[string]any{
												"tweet_results": map[string]any{
													"result": map[string]any{
														"__typename": "Tweet",
														"rest_id":    "s1",
														"legacy":     map[string]any{"full_text": "search result"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	srv := testutil.NewTestServer(testutil.StaticHandler(200, string(searchBody)))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"SearchTimeline": "searchQID"})
	page := c.Search(context.Background(), "golang", nil)
	if !page.Success {
		t.Fatalf("Search: %v", page.Error)
	}
	if len(page.Items) != 1 {
		t.Errorf("want 1 search result, got %d", len(page.Items))
	}
}

func TestClient_Tweet_Wrapper(t *testing.T) {
	respBody, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"create_tweet": map[string]any{
				"tweet_results": map[string]any{
					"result": map[string]any{"rest_id": "newtweet1"},
				},
			},
		},
	})
	srv := testutil.NewTestServer(testutil.StaticHandler(200, string(respBody)))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"CreateTweet": "createQID"})
	id, err := c.Tweet(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("Tweet: %v", err)
	}
	if id != "newtweet1" {
		t.Errorf("Tweet ID: want newtweet1, got %q", id)
	}
}

func TestClient_UploadMedia_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
			return
		}

		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		switch {
		case strings.Contains(bodyStr, "command=INIT"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"media_id_string":"media-wrap-1"}`))
		case strings.Contains(r.Header.Get("Content-Type"), "multipart"):
			w.WriteHeader(http.StatusNoContent)
		case strings.Contains(bodyStr, "command=FINALIZE"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()

	c := newMockClient(t, srv, nil)
	mediaID, err := c.UploadMedia(context.Background(), []byte("small image data"), "image/png", "")
	if err != nil {
		t.Fatalf("UploadMedia: %v", err)
	}
	if mediaID != "media-wrap-1" {
		t.Fatalf("media ID = %q, want %q", mediaID, "media-wrap-1")
	}
}

func TestClient_Reply_Wrapper(t *testing.T) {
	respBody, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"create_tweet": map[string]any{
				"tweet_results": map[string]any{
					"result": map[string]any{"rest_id": "reply1"},
				},
			},
		},
	})
	srv := testutil.NewTestServer(testutil.StaticHandler(200, string(respBody)))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"CreateTweet": "createQID"})
	id, err := c.Reply(context.Background(), "reply text", "parent1")
	if err != nil {
		t.Fatalf("Reply: %v", err)
	}
	if id != "reply1" {
		t.Errorf("Reply ID: want reply1, got %q", id)
	}
}

func TestClient_Like_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, `{"data":{"favorite_tweet":"Done"}}`))
	defer srv.Close()

	c := newMockClient(t, srv, nil)
	if err := c.Like(context.Background(), "liketweet1"); err != nil {
		t.Fatalf("Like: %v", err)
	}
}

func TestClient_Unlike_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, `{"data":{"unfavorite_tweet":"Done"}}`))
	defer srv.Close()

	c := newMockClient(t, srv, nil)
	if err := c.Unlike(context.Background(), "unliketweet1"); err != nil {
		t.Fatalf("Unlike: %v", err)
	}
}

func TestClient_Retweet_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200,
		`{"data":{"create_retweet":{"retweet_results":{"result":{"rest_id":"rt1"}}}}}`))
	defer srv.Close()

	c := newMockClient(t, srv, nil)
	id, err := c.Retweet(context.Background(), "tweet1")
	if err != nil {
		t.Fatalf("Retweet: %v", err)
	}
	if id != "rt1" {
		t.Errorf("Retweet ID: want rt1, got %q", id)
	}
}

func TestClient_Unretweet_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, `{"data":{}}`))
	defer srv.Close()

	c := newMockClient(t, srv, nil)
	if err := c.Unretweet(context.Background(), "tweet2"); err != nil {
		t.Fatalf("Unretweet: %v", err)
	}
}

func TestClient_Bookmark_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, `{"data":{"tweet_bookmark_put":"Done"}}`))
	defer srv.Close()

	c := newMockClient(t, srv, nil)
	if err := c.Bookmark(context.Background(), "tweet3"); err != nil {
		t.Fatalf("Bookmark: %v", err)
	}
}

func TestClient_Unbookmark_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, `{"data":{"tweet_bookmark_delete":"Done"}}`))
	defer srv.Close()

	c := newMockClient(t, srv, nil)
	if err := c.Unbookmark(context.Background(), "tweet4"); err != nil {
		t.Fatalf("Unbookmark: %v", err)
	}
}

func TestClient_Follow_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, `{"id_str":"followed1"}`))
	defer srv.Close()

	c := newMockClient(t, srv, nil)
	if err := c.Follow(context.Background(), "followed1"); err != nil {
		t.Fatalf("Follow: %v", err)
	}
}

func TestClient_Unfollow_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, `{"id_str":"unfollowed1"}`))
	defer srv.Close()

	c := newMockClient(t, srv, nil)
	if err := c.Unfollow(context.Background(), "unfollowed1"); err != nil {
		t.Fatalf("Unfollow: %v", err)
	}
}

func TestClient_ActiveQueryID_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, `{}`))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"HomeTimeline": "hqid"})
	id := c.ActiveQueryID("HomeTimeline")
	if id != "hqid" {
		t.Errorf("ActiveQueryID: want hqid, got %q", id)
	}
}

func TestClient_AllQueryIDs_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, `{}`))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"SearchTimeline": "sqid"})
	ids := c.AllQueryIDs("SearchTimeline")
	found := false
	for _, id := range ids {
		if id == "sqid" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("AllQueryIDs should include injected ID, got %v", ids)
	}
}

func TestClient_RefreshQueryIDs_Wrapper(t *testing.T) {
	// RefreshQueryIDs delegates to the internal client; verify it exists and
	// accepts a context. Use an already-cancelled context so the underlying
	// network scrape exits immediately without making real HTTP calls.
	srv := testutil.NewTestServer(testutil.StaticHandler(200, `{}`))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled immediately — scrape exits without network I/O
	c := newMockClient(t, srv, nil)
	c.RefreshQueryIDs(ctx) // must not panic or hang
}

func TestClient_GetUserTweets_Wrapper(t *testing.T) {
	userTweetsBody, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"user": map[string]any{
				"result": map[string]any{
					"timeline": map[string]any{
						"timeline": map[string]any{
							"instructions": []any{
								map[string]any{
									"type": "TimelineAddEntries",
									"entries": []any{
										map[string]any{
											"entryId":   "tweet-ut1",
											"sortIndex": "1",
											"content": map[string]any{
												"entryType": "TimelineTimelineItem",
												"itemContent": map[string]any{
													"__typename": "TimelineTweet",
													"tweet_results": map[string]any{
														"result": map[string]any{
															"__typename": "Tweet",
															"rest_id":    "ut1",
															"legacy": map[string]any{
																"full_text":           "user tweet 1",
																"created_at":          "",
																"conversation_id_str": "ut1",
																"reply_count":         0,
																"retweet_count":       0,
																"favorite_count":      0,
																"user_id_str":         "u1",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	srv := testutil.NewTestServer(testutil.StaticHandler(200, string(userTweetsBody)))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"UserTweets": "utQID"})
	result, err := c.GetUserTweets(context.Background(), "u1", nil)
	if err != nil {
		t.Fatalf("GetUserTweets: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("GetUserTweets: expected success")
	}
	if len(result.Items) != 1 {
		t.Errorf("want 1 tweet, got %d", len(result.Items))
	}
}

func TestClient_GetUserTweetsPaged_Wrapper(t *testing.T) {
	userTweetsBody, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"user": map[string]any{
				"result": map[string]any{
					"timeline": map[string]any{
						"timeline": map[string]any{
							"instructions": []any{
								map[string]any{
									"type": "TimelineAddEntries",
									"entries": []any{
										map[string]any{
											"entryId":   "tweet-up1",
											"sortIndex": "1",
											"content": map[string]any{
												"entryType": "TimelineTimelineItem",
												"itemContent": map[string]any{
													"__typename": "TimelineTweet",
													"tweet_results": map[string]any{
														"result": map[string]any{
															"__typename": "Tweet",
															"rest_id":    "up1",
															"legacy": map[string]any{
																"full_text":           "paged tweet",
																"created_at":          "",
																"conversation_id_str": "up1",
																"reply_count":         0,
																"retweet_count":       0,
																"favorite_count":      0,
																"user_id_str":         "u2",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	srv := testutil.NewTestServer(testutil.StaticHandler(200, string(userTweetsBody)))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"UserTweets": "utQID"})
	page, err := c.GetUserTweetsPaged(context.Background(), "u2", "")
	if err != nil {
		t.Fatalf("GetUserTweetsPaged: %v", err)
	}
	if page == nil || !page.Success {
		t.Fatalf("GetUserTweetsPaged: expected success")
	}
	if len(page.Items) != 1 {
		t.Errorf("want 1 tweet, got %d", len(page.Items))
	}
}

func TestClient_GetAllSearchResults_Wrapper(t *testing.T) {
	searchBody, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"search_by_raw_query": map[string]any{
				"search_timeline": map[string]any{
					"timeline": map[string]any{
						"instructions": []any{
							map[string]any{
								"entries": []any{
									map[string]any{
										"entryId": "tweet-as1",
										"content": map[string]any{
											"itemContent": map[string]any{
												"tweet_results": map[string]any{
													"result": map[string]any{
														"__typename": "Tweet",
														"rest_id":    "as1",
														"legacy":     map[string]any{"full_text": "all search"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	srv := testutil.NewTestServer(testutil.StaticHandler(200, string(searchBody)))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"SearchTimeline": "searchQID"})
	result := c.GetAllSearchResults(context.Background(), "golang", nil)
	if !result.Success {
		t.Fatalf("GetAllSearchResults: %v", result.Error)
	}
	if len(result.Items) != 1 {
		t.Errorf("want 1 item, got %d", len(result.Items))
	}
}

func TestClient_GetCurrentUser_Wrapper(t *testing.T) {
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id_str":"me1","screen_name":"myhandle","name":"My Name"}`))
	}))
	defer srv.Close()

	c := newMockClient(t, srv, nil)
	user, err := c.GetCurrentUser(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentUser: %v", err)
	}
	if user == nil {
		t.Fatal("GetCurrentUser: want non-nil result")
	}
	if user.ID != "me1" {
		t.Errorf("ID: want me1, got %q", user.ID)
	}
}

func TestClient_GetReplies_Wrapper(t *testing.T) {
	threadBody, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"threaded_conversation_with_injections_v2": map[string]any{
				"instructions": []any{
					map[string]any{
						"type": "TimelineAddEntries",
						"entries": []any{
							map[string]any{
								"entryId":   "tweet-r1",
								"sortIndex": "2",
								"content": map[string]any{
									"entryType": "TimelineTimelineItem",
									"itemContent": map[string]any{
										"__typename": "TimelineTweet",
										"tweet_results": map[string]any{
											"result": map[string]any{
												"__typename": "Tweet",
												"rest_id":    "r1",
												"legacy": map[string]any{
													"full_text":           "reply tweet",
													"created_at":          "",
													"conversation_id_str": "root1",
													"reply_count":         0,
													"retweet_count":       0,
													"favorite_count":      0,
													"user_id_str":         "u1",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	srv := testutil.NewTestServer(testutil.StaticHandler(200, string(threadBody)))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"TweetDetail": "threadQID"})
	result, err := c.GetReplies(context.Background(), "root1", nil)
	if err != nil {
		t.Fatalf("GetReplies: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("GetReplies: expected success")
	}
}

func TestClient_GetThread_Wrapper(t *testing.T) {
	threadBody, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"threaded_conversation_with_injections_v2": map[string]any{
				"instructions": []any{
					map[string]any{
						"type": "TimelineAddEntries",
						"entries": []any{
							map[string]any{
								"entryId":   "tweet-th1",
								"sortIndex": "1",
								"content": map[string]any{
									"entryType": "TimelineTimelineItem",
									"itemContent": map[string]any{
										"__typename": "TimelineTweet",
										"tweet_results": map[string]any{
											"result": map[string]any{
												"__typename": "Tweet",
												"rest_id":    "th1",
												"legacy": map[string]any{
													"full_text":           "thread tweet",
													"created_at":          "",
													"conversation_id_str": "th1",
													"reply_count":         0,
													"retweet_count":       0,
													"favorite_count":      0,
													"user_id_str":         "u1",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	srv := testutil.NewTestServer(testutil.StaticHandler(200, string(threadBody)))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"TweetDetail": "threadQID"})
	tweets, err := c.GetThread(context.Background(), "th1", nil)
	if err != nil {
		t.Fatalf("GetThread: %v", err)
	}
	if len(tweets) == 0 {
		t.Error("GetThread: want at least 1 tweet")
	}
}

func TestClient_GetFollowing_Wrapper(t *testing.T) {
	followBody, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"user": map[string]any{
				"result": map[string]any{
					"timeline": map[string]any{
						"timeline": map[string]any{
							"instructions": []any{
								map[string]any{
									"type": "TimelineAddEntries",
									"entries": []any{
										map[string]any{
											"entryId":   "user-f1",
											"sortIndex": "1",
											"content": map[string]any{
												"entryType": "TimelineTimelineItem",
												"itemContent": map[string]any{
													"__typename": "TimelineUser",
													"user_results": map[string]any{
														"result": map[string]any{
															"__typename": "User",
															"rest_id":    "f1",
															"legacy": map[string]any{
																"screen_name": "following1",
																"name":        "Following One",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	srv := testutil.NewTestServer(testutil.StaticHandler(200, string(followBody)))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"Following": "followQID"})
	result, err := c.GetFollowing(context.Background(), "u1", nil)
	if err != nil {
		t.Fatalf("GetFollowing: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("GetFollowing: expected success")
	}
}

func TestClient_GetFollowers_Wrapper(t *testing.T) {
	followBody, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"user": map[string]any{
				"result": map[string]any{
					"timeline": map[string]any{
						"timeline": map[string]any{
							"instructions": []any{
								map[string]any{
									"type": "TimelineAddEntries",
									"entries": []any{
										map[string]any{
											"entryId":   "user-fr1",
											"sortIndex": "1",
											"content": map[string]any{
												"entryType": "TimelineTimelineItem",
												"itemContent": map[string]any{
													"__typename": "TimelineUser",
													"user_results": map[string]any{
														"result": map[string]any{
															"__typename": "User",
															"rest_id":    "fr1",
															"legacy": map[string]any{
																"screen_name": "follower1",
																"name":        "Follower One",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	srv := testutil.NewTestServer(testutil.StaticHandler(200, string(followBody)))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"Followers": "followerQID"})
	result, err := c.GetFollowers(context.Background(), "u1", nil)
	if err != nil {
		t.Fatalf("GetFollowers: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("GetFollowers: expected success")
	}
}

func TestClient_GetLikes_Wrapper(t *testing.T) {
	likesBody, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"user": map[string]any{
				"result": map[string]any{
					"timeline": map[string]any{
						"timeline": map[string]any{
							"instructions": []any{
								map[string]any{
									"entries": []any{
										map[string]any{
											"entryId": "tweet-lk1",
											"content": map[string]any{
												"itemContent": map[string]any{
													"tweet_results": map[string]any{
														"result": map[string]any{
															"__typename": "Tweet",
															"rest_id":    "lk1",
															"legacy":     map[string]any{"full_text": "liked tweet"},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	callCount := 0
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			w.Write([]byte(`{"id_str":"myuid","screen_name":"me","name":"Me"}`))
			return
		}
		w.Write(likesBody)
	}))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"Likes": "likesQID"})
	result := c.GetLikes(context.Background(), nil)
	if !result.Success {
		t.Fatalf("GetLikes: %v", result.Error)
	}
}

func TestClient_GetUserIDByUsername_Wrapper(t *testing.T) {
	userJSON := `{"data":{"user":{"result":{"__typename":"User","rest_id":"uid42","legacy":{"screen_name":"testuser","name":"Test User"}}}}}`
	srv := testutil.NewTestServer(testutil.StaticHandler(200, userJSON))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{
		"UserByScreenName": "xc8f1g7BYqr6VTzTbvNlGw",
	})
	id, err := c.GetUserIDByUsername(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("GetUserIDByUsername: %v", err)
	}
	if id != "uid42" {
		t.Errorf("ID: want uid42, got %q", id)
	}
}

func TestClient_GetUserAboutAccount_Wrapper(t *testing.T) {
	aboutJSON := `{"data":{"user_result_by_screen_name":{"result":{"about_profile":{"user_id":"777","screen_name":"aboutuser","name":"About User","description":"bio","followers_count":10,"friends_count":5}}}}}`
	srv := testutil.NewTestServer(testutil.StaticHandler(200, aboutJSON))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{
		"AboutAccountQuery": "zs_jFPFT78rBpXv9Z3U2YQ",
	})
	user, err := c.GetUserAboutAccount(context.Background(), "aboutuser")
	if err != nil {
		t.Fatalf("GetUserAboutAccount: %v", err)
	}
	if user == nil {
		t.Fatal("GetUserAboutAccount: want non-nil result")
	}
	if user.ID != "777" {
		t.Errorf("ID: want 777, got %q", user.ID)
	}
}

func TestClient_GetBookmarkFolderTimeline_Wrapper(t *testing.T) {
	folderBody, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"bookmark_collection_timeline": map[string]any{
				"timeline": map[string]any{
					"instructions": []any{
						map[string]any{
							"entries": []any{
								map[string]any{
									"entryId": "tweet-bf1",
									"content": map[string]any{
										"itemContent": map[string]any{
											"tweet_results": map[string]any{
												"result": map[string]any{
													"__typename": "Tweet",
													"rest_id":    "bf1",
													"legacy":     map[string]any{"full_text": "folder tweet"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	srv := testutil.NewTestServer(testutil.StaticHandler(200, string(folderBody)))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"BookmarkFolderTimeline": "folderQID"})
	result := c.GetBookmarkFolderTimeline(context.Background(), &bird.BookmarkFolderOptions{FolderID: "folder1"})
	if !result.Success {
		t.Fatalf("GetBookmarkFolderTimeline: %v", result.Error)
	}
}

func TestClient_GetOwnedLists_Wrapper(t *testing.T) {
	listResp := `{"data":{"user":{"result":{"timeline":{"timeline":{"instructions":[{"entries":[]}]}}}}}}`
	callCount := 0
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			w.Write([]byte(`{"id_str":"myuid","screen_name":"me","name":"Me"}`))
			return
		}
		w.Write([]byte(listResp))
	}))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"ListOwnerships": "ownedQID"})
	result, err := c.GetOwnedLists(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetOwnedLists: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("GetOwnedLists: expected success")
	}
}

func TestClient_GetListMemberships_Wrapper(t *testing.T) {
	listResp := `{"data":{"user":{"result":{"timeline":{"timeline":{"instructions":[{"entries":[]}]}}}}}}`
	callCount := 0
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			w.Write([]byte(`{"id_str":"myuid","screen_name":"me","name":"Me"}`))
			return
		}
		w.Write([]byte(listResp))
	}))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"ListMemberships": "membQID"})
	result, err := c.GetListMemberships(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetListMemberships: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("GetListMemberships: expected success")
	}
}

func TestClient_GetNews_Wrapper(t *testing.T) {
	newsBody := `{"data":{"timeline":{"timeline":{"instructions":[{"type":"TimelineAddEntries","entries":[]}]}}}}`
	srv := testutil.NewTestServer(testutil.StaticHandler(200, newsBody))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"GenericTimelineById": "newsQID"})
	items, err := c.GetNews(context.Background(), &bird.NewsOptions{Tabs: []string{"news"}})
	if err != nil {
		t.Fatalf("GetNews: %v", err)
	}
	_ = items
}

func TestClient_GetListTimeline_Wrapper(t *testing.T) {
	listBody, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"list": map[string]any{
				"tweets_timeline": map[string]any{
					"timeline": map[string]any{
						"instructions": []any{
							map[string]any{
								"type": "TimelineAddEntries",
								"entries": []any{
									map[string]any{
										"entryId":   "tweet-lt1",
										"sortIndex": "1",
										"content": map[string]any{
											"entryType": "TimelineTimelineItem",
											"itemContent": map[string]any{
												"__typename": "TimelineTweet",
												"tweet_results": map[string]any{
													"result": map[string]any{
														"__typename": "Tweet",
														"rest_id":    "lt1",
														"legacy": map[string]any{
															"full_text":           "list tweet",
															"created_at":          "",
															"conversation_id_str": "lt1",
															"reply_count":         0,
															"retweet_count":       0,
															"favorite_count":      0,
															"user_id_str":         "u1",
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	srv := testutil.NewTestServer(testutil.StaticHandler(200, string(listBody)))
	defer srv.Close()

	c := newMockClient(t, srv, map[string]string{"ListLatestTweetsTimeline": "listQID"})
	result, err := c.GetListTimeline(context.Background(), "list1", nil)
	if err != nil {
		t.Fatalf("GetListTimeline: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("GetListTimeline: expected success")
	}
}
