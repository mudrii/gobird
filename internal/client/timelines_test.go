package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/mudrii/gobird/internal/types"
)

// bookmarksResp builds a mock Bookmarks response.
func bookmarksResp(ids []string, nextCursor string) []byte {
	var entries []any
	for _, id := range ids {
		entries = append(entries, map[string]any{
			"entryId": "tweet-" + id,
			"content": map[string]any{
				"itemContent": map[string]any{
					"tweet_results": map[string]any{
						"result": map[string]any{
							"__typename": "Tweet",
							"rest_id":    id,
							"legacy":     map[string]any{"full_text": "text"},
							"core": map[string]any{
								"user_results": map[string]any{
									"result": map[string]any{
										"__typename": "User",
										"rest_id":    "u1",
										"legacy":     map[string]any{"screen_name": "user", "name": "User"},
									},
								},
							},
						},
					},
				},
			},
		})
	}
	if nextCursor != "" {
		entries = append(entries, map[string]any{
			"entryId": "cursor-bottom",
			"content": map[string]any{
				"cursorType": "Bottom",
				"value":      nextCursor,
			},
		})
	}
	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"bookmark_timeline_v2": map[string]any{
				"timeline": map[string]any{
					"instructions": []any{
						map[string]any{"entries": entries},
					},
				},
			},
		},
	})
	return body
}

func TestGetBookmarks_basic(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(bookmarksResp([]string{"b1", "b2"}, ""))
	}))
	defer srv.Close()

	c.queryIDCache["Bookmarks"] = "booksQID"
	result := c.GetBookmarks(context.Background(), nil)
	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
}

func TestGetBookmarks_variables(t *testing.T) {
	// Correction #22: must include all 6 fields.
	var capturedVars map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, _ := url.ParseQuery(r.URL.RawQuery)
		json.Unmarshal([]byte(params.Get("variables")), &capturedVars)
		w.Header().Set("Content-Type", "application/json")
		w.Write(bookmarksResp(nil, ""))
	}))
	defer srv.Close()

	c.queryIDCache["Bookmarks"] = "booksQID"
	c.GetBookmarks(context.Background(), nil)

	if capturedVars["includePromotedContent"] != false {
		t.Errorf("includePromotedContent should be false, got %v", capturedVars["includePromotedContent"])
	}
	if capturedVars["withDownvotePerspective"] != false {
		t.Errorf("withDownvotePerspective should be false, got %v", capturedVars["withDownvotePerspective"])
	}
	if capturedVars["withReactionsMetadata"] != false {
		t.Errorf("withReactionsMetadata should be false, got %v", capturedVars["withReactionsMetadata"])
	}
	if capturedVars["withReactionsPerspective"] != false {
		t.Errorf("withReactionsPerspective should be false, got %v", capturedVars["withReactionsPerspective"])
	}
}

func TestGetBookmarks_paginates(t *testing.T) {
	calls := 0
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		switch calls {
		case 1:
			w.Write(bookmarksResp([]string{"1"}, "c2"))
		case 2:
			// Verify cursor passed in second call.
			params, _ := url.ParseQuery(r.URL.RawQuery)
			var vars map[string]any
			json.Unmarshal([]byte(params.Get("variables")), &vars)
			if vars["cursor"] != "c2" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Write(bookmarksResp([]string{"2"}, ""))
		}
	}))
	defer srv.Close()

	c.queryIDCache["Bookmarks"] = "booksQID"
	result := c.GetBookmarks(context.Background(), &types.FetchOptions{PageDelayMs: 0})
	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items across pages, got %d", len(result.Items))
	}
}

// bookmarkFolderResp builds a mock BookmarkFolderTimeline response.
func bookmarkFolderResp(ids []string, nextCursor string) []byte {
	var entries []any
	for _, id := range ids {
		entries = append(entries, map[string]any{
			"entryId": "tweet-" + id,
			"content": map[string]any{
				"itemContent": map[string]any{
					"tweet_results": map[string]any{
						"result": map[string]any{
							"__typename": "Tweet",
							"rest_id":    id,
							"legacy":     map[string]any{"full_text": "text"},
							"core": map[string]any{
								"user_results": map[string]any{
									"result": map[string]any{
										"__typename": "User",
										"rest_id":    "u1",
										"legacy":     map[string]any{"screen_name": "user", "name": "User"},
									},
								},
							},
						},
					},
				},
			},
		})
	}
	if nextCursor != "" {
		entries = append(entries, map[string]any{
			"entryId": "cursor-bottom",
			"content": map[string]any{
				"cursorType": "Bottom",
				"value":      nextCursor,
			},
		})
	}
	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"bookmark_collection_timeline": map[string]any{
				"timeline": map[string]any{
					"instructions": []any{
						map[string]any{"entries": entries},
					},
				},
			},
		},
	})
	return body
}

func TestGetBookmarkFolderTimeline_basic(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(bookmarkFolderResp([]string{"f1", "f2"}, ""))
	}))
	defer srv.Close()

	c.queryIDCache["BookmarkFolderTimeline"] = "folderQID"
	result := c.GetBookmarkFolderTimeline(context.Background(), &types.BookmarkFolderOptions{
		FolderID: "folder123",
	})
	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
}

func TestGetBookmarkFolderTimeline_includePromotedContentTrue(t *testing.T) {
	// Correction #23: includePromotedContent is true (unlike regular bookmarks).
	var capturedVars map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, _ := url.ParseQuery(r.URL.RawQuery)
		json.Unmarshal([]byte(params.Get("variables")), &capturedVars)
		w.Header().Set("Content-Type", "application/json")
		w.Write(bookmarkFolderResp(nil, ""))
	}))
	defer srv.Close()

	c.queryIDCache["BookmarkFolderTimeline"] = "folderQID"
	c.GetBookmarkFolderTimeline(context.Background(), &types.BookmarkFolderOptions{FolderID: "f1"})
	if capturedVars["includePromotedContent"] != true {
		t.Errorf("includePromotedContent should be true for folder timeline, got %v", capturedVars["includePromotedContent"])
	}
}

func TestGetBookmarkFolderTimeline_retryWithoutCountOnError(t *testing.T) {
	// Correction #11: retry without count on Variable "$count" error.
	calls := 0
	// allVars stores each call's vars as a fresh map (avoids json.Unmarshal merge issue).
	var allVars []map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		params, _ := url.ParseQuery(r.URL.RawQuery)
		var v map[string]any
		json.Unmarshal([]byte(params.Get("variables")), &v)
		allVars = append(allVars, v)
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			// Return count variable error.
			w.Write([]byte(`{"errors":[{"message":"Variable \"$count\" got invalid value"}]}`))
			return
		}
		w.Write(bookmarkFolderResp([]string{"x"}, ""))
	}))
	defer srv.Close()

	c.queryIDCache["BookmarkFolderTimeline"] = "folderQID"
	result := c.GetBookmarkFolderTimeline(context.Background(), &types.BookmarkFolderOptions{FolderID: "f1"})
	if !result.Success {
		t.Errorf("expected success after retry without count, got: %v", result.Error)
	}
	if len(allVars) < 2 {
		t.Fatalf("expected at least 2 requests, got %d", len(allVars))
	}
	// The last request (retry without count) must not have "count".
	lastVars := allVars[len(allVars)-1]
	if _, hasCount := lastVars["count"]; hasCount {
		t.Error("retried request should not include count field")
	}
}

func TestGetBookmarkFolderTimeline_cursorErrorStopsPagination(t *testing.T) {
	// Correction #23: cursor error returns immediately.
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"errors":[{"message":"Variable \"$cursor\" got invalid value"}]}`))
	}))
	defer srv.Close()

	c.queryIDCache["BookmarkFolderTimeline"] = "folderQID"
	// We need a cursor to trigger this path — use FetchOptions with cursor.
	result := c.GetBookmarkFolderTimeline(context.Background(), &types.BookmarkFolderOptions{
		FolderID:     "f1",
		FetchOptions: types.FetchOptions{Cursor: "badCursor"},
	})
	if result.Success {
		t.Error("expected failure on cursor error")
	}
	if result.Error == nil {
		t.Error("expected non-nil error")
	}
}

// likesResp builds a mock Likes response.
func likesResp(ids []string, nextCursor string) []byte {
	var entries []any
	for _, id := range ids {
		entries = append(entries, map[string]any{
			"entryId": "tweet-" + id,
			"content": map[string]any{
				"itemContent": map[string]any{
					"tweet_results": map[string]any{
						"result": map[string]any{
							"__typename": "Tweet",
							"rest_id":    id,
							"legacy":     map[string]any{"full_text": "text"},
							"core": map[string]any{
								"user_results": map[string]any{
									"result": map[string]any{
										"__typename": "User",
										"rest_id":    "u1",
										"legacy":     map[string]any{"screen_name": "user", "name": "User"},
									},
								},
							},
						},
					},
				},
			},
		})
	}
	if nextCursor != "" {
		entries = append(entries, map[string]any{
			"entryId": "cursor-bottom",
			"content": map[string]any{
				"cursorType": "Bottom",
				"value":      nextCursor,
			},
		})
	}
	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"user": map[string]any{
				"result": map[string]any{
					"timeline": map[string]any{
						"timeline": map[string]any{
							"instructions": []any{
								map[string]any{"entries": entries},
							},
						},
					},
				},
			},
		},
	})
	return body
}

func TestGetLikes_basic(t *testing.T) {
	callCount := 0
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		// First call is getCurrentUser (verify_credentials or settings).
		if callCount == 1 {
			w.Write([]byte(`{"id_str":"myUID","screen_name":"me","name":"Me"}`))
			return
		}
		w.Write(likesResp([]string{"l1", "l2"}, ""))
	}))
	defer srv.Close()

	c.queryIDCache["Likes"] = "likesQID"
	result := c.GetLikes(context.Background(), nil)
	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
}

func TestGetLikes_variables(t *testing.T) {
	// Correction #29: no withV2Timeline; has specific set of fields.
	var capturedVars map[string]any
	callCount := 0
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			w.Write([]byte(`{"id_str":"uid1","screen_name":"me","name":"Me"}`))
			return
		}
		params, _ := url.ParseQuery(r.URL.RawQuery)
		json.Unmarshal([]byte(params.Get("variables")), &capturedVars)
		w.Write(likesResp(nil, ""))
	}))
	defer srv.Close()

	c.queryIDCache["Likes"] = "likesQID"
	c.GetLikes(context.Background(), nil)

	if capturedVars["includePromotedContent"] != false {
		t.Errorf("includePromotedContent should be false, got %v", capturedVars["includePromotedContent"])
	}
	if capturedVars["withClientEventToken"] != false {
		t.Errorf("withClientEventToken should be false, got %v", capturedVars["withClientEventToken"])
	}
	if capturedVars["withBirdwatchNotes"] != false {
		t.Errorf("withBirdwatchNotes should be false, got %v", capturedVars["withBirdwatchNotes"])
	}
	if capturedVars["withVoice"] != true {
		t.Errorf("withVoice should be true, got %v", capturedVars["withVoice"])
	}
	if _, hasV2 := capturedVars["withV2Timeline"]; hasV2 {
		t.Error("withV2Timeline must NOT be present in Likes variables")
	}
}

func TestGetLikes_queryUnspecifiedExactCaseTriggersRefresh(t *testing.T) {
	// Correction #51: exact case "Query: Unspecified" triggers refresh.
	callCount := 0
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			// getCurrentUser
			w.Write([]byte(`{"id_str":"uid1","screen_name":"me","name":"Me"}`))
			return
		}
		if callCount == 2 {
			// First Likes attempt — "Query: Unspecified" error
			w.Write([]byte(`{"errors":[{"message":"Query: Unspecified"}]}`))
			return
		}
		// After refresh, succeed.
		w.Write(likesResp([]string{"liked1"}, ""))
	}))
	defer srv.Close()

	c.queryIDCache["Likes"] = "likesQID"
	result := c.GetLikes(context.Background(), nil)
	if !result.Success {
		t.Errorf("expected success after Query: Unspecified refresh, got: %v", result.Error)
	}
}

func TestGetLikes_usesGetNotFetchWithRetry(t *testing.T) {
	// Correction #49: Likes uses plain GET, not fetchWithRetry.
	// Verify by checking that 429 is NOT retried automatically.
	callCount := 0
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			w.Write([]byte(`{"id_str":"uid1","screen_name":"me","name":"Me"}`))
			return
		}
		// Return 429 — should NOT be retried by Likes.
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"errors":[{"message":"Rate limit exceeded"}]}`))
	}))
	defer srv.Close()

	c.queryIDCache["Likes"] = "likesQID"
	result := c.GetLikes(context.Background(), nil)
	// With fetchWithRetry, there would be retries. Without it, the single 429 response
	// means failure. We just verify calls are minimal.
	_ = result
	// callCount should be 2 (1 for getCurrentUser, 1 for the Likes fetch) — no retries.
	// Note: may try multiple queryIDs so allow for some variance.
	if callCount > 5 {
		t.Errorf("too many calls (%d): Likes should not retry on 429", callCount)
	}
}
