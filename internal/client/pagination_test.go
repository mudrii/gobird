package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/mudrii/gobird/internal/testutil"
	"github.com/mudrii/gobird/internal/types"
)

// makeTweet returns a minimal TweetData with the given ID.
func makeTweet(id string) types.TweetData {
	return types.TweetData{ID: id}
}

func TestPaginateInline_SinglePage(t *testing.T) {
	calls := 0
	fetch := func(_ context.Context, cursor string) inlinePageResult {
		calls++
		return inlinePageResult{
			tweets:     []types.TweetData{makeTweet("1"), makeTweet("2")},
			nextCursor: "",
			success:    true,
		}
	}

	result := paginateInline(context.Background(), types.FetchOptions{}, 0, fetch)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
	if calls != 1 {
		t.Errorf("expected 1 fetch call, got %d", calls)
	}
}

func TestPaginateInline_MultiPage(t *testing.T) {
	pages := []inlinePageResult{
		{tweets: []types.TweetData{makeTweet("1")}, nextCursor: "c2", success: true},
		{tweets: []types.TweetData{makeTweet("2")}, nextCursor: "c3", success: true},
		{tweets: []types.TweetData{makeTweet("3")}, nextCursor: "", success: true},
	}
	idx := 0
	fetch := func(_ context.Context, cursor string) inlinePageResult {
		p := pages[idx]
		idx++
		return p
	}

	result := paginateInline(context.Background(), types.FetchOptions{}, 0, fetch)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(result.Items))
	}
}

func TestPaginateInline_StopsOnCursorUnchanged(t *testing.T) {
	calls := 0
	fetch := func(_ context.Context, cursor string) inlinePageResult {
		calls++
		// Always return same cursor to trigger stop.
		return inlinePageResult{
			tweets:     []types.TweetData{makeTweet("1")},
			nextCursor: "same",
			success:    true,
		}
	}

	opts := types.FetchOptions{Cursor: "same"}
	result := paginateInline(context.Background(), opts, 0, fetch)
	if !result.Success {
		t.Fatalf("expected success")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (stopped on same cursor), got %d", calls)
	}
}

func TestPaginateInline_StopsOnEmptyTweets(t *testing.T) {
	calls := 0
	fetch := func(_ context.Context, cursor string) inlinePageResult {
		calls++
		return inlinePageResult{
			tweets:     []types.TweetData{},
			nextCursor: "next",
			success:    true,
		}
	}

	result := paginateInline(context.Background(), types.FetchOptions{}, 0, fetch)
	if !result.Success {
		t.Fatalf("expected success")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (stopped on empty tweets), got %d", calls)
	}
}

func TestPaginateInline_StopsOnAddedZero(t *testing.T) {
	// Return the same tweet every page — added will be 0 on second call.
	calls := 0
	fetch := func(_ context.Context, cursor string) inlinePageResult {
		calls++
		nc := "next"
		if cursor == "next" {
			nc = "next2"
		}
		return inlinePageResult{
			tweets:     []types.TweetData{makeTweet("dup")},
			nextCursor: nc,
			success:    true,
		}
	}

	result := paginateInline(context.Background(), types.FetchOptions{}, 0, fetch)
	if !result.Success {
		t.Fatalf("expected success")
	}
	// First call adds the tweet; second call sees it's a dup so added=0 → stop.
	if calls != 2 {
		t.Errorf("expected 2 calls (stop on added=0), got %d", calls)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 unique item, got %d", len(result.Items))
	}
}

func TestPaginateInline_StopsOnMaxPages(t *testing.T) {
	calls := 0
	fetch := func(_ context.Context, cursor string) inlinePageResult {
		calls++
		return inlinePageResult{
			tweets:     []types.TweetData{makeTweet("id-" + string(rune('a'+calls)))},
			nextCursor: "next",
			success:    true,
		}
	}

	opts := types.FetchOptions{MaxPages: 2}
	result := paginateInline(context.Background(), opts, 0, fetch)
	if !result.Success {
		t.Fatalf("expected success")
	}
	if calls != 2 {
		t.Errorf("expected 2 calls (MaxPages=2), got %d", calls)
	}
}

func TestPaginateInline_StopsOnLimit(t *testing.T) {
	calls := 0
	fetch := func(_ context.Context, cursor string) inlinePageResult {
		calls++
		// Return 3 unique tweets per page.
		offset := (calls - 1) * 3
		tweets := []types.TweetData{
			makeTweet(string(rune('a' + offset))),
			makeTweet(string(rune('b' + offset))),
			makeTweet(string(rune('c' + offset))),
		}
		return inlinePageResult{
			tweets:     tweets,
			nextCursor: "next",
			success:    true,
		}
	}

	opts := types.FetchOptions{Limit: 3}
	result := paginateInline(context.Background(), opts, 0, fetch)
	if !result.Success {
		t.Fatalf("expected success")
	}
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items (Limit=3), got %d", len(result.Items))
	}
}

func TestPaginateInline_ErrorOnFirstPage_NoItems(t *testing.T) {
	fetch := func(_ context.Context, cursor string) inlinePageResult {
		return inlinePageResult{success: false, err: errors.New("API error")}
	}

	result := paginateInline(context.Background(), types.FetchOptions{}, 0, fetch)
	if result.Success {
		t.Fatal("expected failure")
	}
	if result.Error == nil {
		t.Fatal("expected non-nil error")
	}
}

func TestPaginateInline_ErrorAfterItems_ReturnsPartial(t *testing.T) {
	calls := 0
	fetch := func(_ context.Context, cursor string) inlinePageResult {
		calls++
		if calls == 1 {
			return inlinePageResult{
				tweets:     []types.TweetData{makeTweet("1")},
				nextCursor: "c2",
				success:    true,
			}
		}
		return inlinePageResult{success: false, err: errors.New("network error")}
	}

	result := paginateInline(context.Background(), types.FetchOptions{}, 0, fetch)
	if result.Success {
		t.Fatal("expected failure after partial success")
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 partial item, got %d", len(result.Items))
	}
	if result.Error == nil {
		t.Fatal("expected non-nil error")
	}
}

func TestPaginateInline_DeduplicatesByID(t *testing.T) {
	calls := 0
	fetch := func(_ context.Context, cursor string) inlinePageResult {
		calls++
		if calls == 1 {
			return inlinePageResult{
				tweets:     []types.TweetData{makeTweet("a"), makeTweet("b")},
				nextCursor: "c2",
				success:    true,
			}
		}
		return inlinePageResult{
			tweets:     []types.TweetData{makeTweet("b"), makeTweet("c")},
			nextCursor: "",
			success:    true,
		}
	}

	result := paginateInline(context.Background(), types.FetchOptions{}, 0, fetch)
	if !result.Success {
		t.Fatalf("expected success")
	}
	// "b" appears on both pages but should only be counted once.
	if len(result.Items) != 3 {
		t.Errorf("expected 3 unique items, got %d", len(result.Items))
	}
}

// threadedConvResponse builds a threaded_conversation_with_injections_v2 JSON
// response containing tweetIDs as tweet entries plus an optional bottom cursor.
func threadedConvResponse(tweetIDs []string, bottomCursor string) string {
	entries := ""
	sortIdx := len(tweetIDs) + 1
	for i, id := range tweetIDs {
		if i > 0 {
			entries += ","
		}
		entries += fmt.Sprintf(`{
			"entryId":"tweet-%s",
			"sortIndex":"%d",
			"content":{
				"entryType":"TimelineTimelineItem",
				"itemContent":{
					"__typename":"TimelineTweet",
					"tweet_results":{
						"result":{
							"__typename":"Tweet",
							"rest_id":"%s",
							"legacy":{
								"full_text":"tweet %s",
								"created_at":"",
								"conversation_id_str":"%s",
								"reply_count":0,"retweet_count":0,"favorite_count":0,
								"user_id_str":"1"
							}
						}
					}
				}
			}
		}`, id, sortIdx-i, id, id, id)
	}
	if bottomCursor != "" {
		if entries != "" {
			entries += ","
		}
		entries += fmt.Sprintf(`{
			"entryId":"cursor-bottom",
			"sortIndex":"0",
			"content":{
				"entryType":"TimelineTimelineCursor",
				"cursorType":"Bottom",
				"value":%q
			}
		}`, bottomCursor)
	}
	return fmt.Sprintf(`{
		"data":{
			"threaded_conversation_with_injections_v2":{
				"instructions":[{
					"type":"TimelineAddEntries",
					"entries":[%s]
				}]
			}
		}
	}`, entries)
}

func TestPaginateCursor_StopsOnEmptyCursor(t *testing.T) {
	// Server returns tweets with no bottom cursor → should stop after 1 page.
	calls := 0
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(threadedConvResponse([]string{"t1", "t2"}, "")))
	}))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{"TweetDetail": "_NvJCnIjOW__EP5-RF197A"})
	result, err := c.GetReplies(context.Background(), "t1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if calls != 1 {
		t.Errorf("want 1 call when cursor is empty, got %d", calls)
	}
}

func TestPaginateCursor_StopsOnUnchangedCursor(t *testing.T) {
	// Server always returns the same cursor value → pagination stops.
	calls := 0
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(threadedConvResponse([]string{"t1"}, "same-cursor")))
	}))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{"TweetDetail": "_NvJCnIjOW__EP5-RF197A"})
	// Pass the same cursor the server will return → stops immediately.
	result, err := c.GetReplies(context.Background(), "t1", &types.ThreadOptions{
		FetchOptions: types.FetchOptions{Cursor: "same-cursor"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success")
	}
	if calls != 1 {
		t.Errorf("want 1 call when cursor unchanged, got %d", calls)
	}
}

func TestPaginateCursor_ZeroItemsDoesNotStop(t *testing.T) {
	// paginateCursor does NOT stop on zero items — only on cursor progress.
	calls := 0
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		// Each response has no tweets but has a new cursor — stops only when cursor repeats.
		cursor := ""
		if calls < 3 {
			cursor = fmt.Sprintf("cursor-%d", calls)
		}
		_, _ = w.Write([]byte(threadedConvResponse([]string{}, cursor)))
	}))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{"TweetDetail": "_NvJCnIjOW__EP5-RF197A"})
	_, err := c.GetReplies(context.Background(), "t1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have fetched more than 1 page since cursor kept changing with zero items.
	if calls < 2 {
		t.Errorf("paginateCursor should continue on 0-item pages with new cursor; got %d calls", calls)
	}
}

func TestPaginateCursor_RespectsMaxPages(t *testing.T) {
	calls := 0
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		id := fmt.Sprintf("t%d", calls)
		cursor := fmt.Sprintf("cursor-%d", calls)
		_, _ = w.Write([]byte(threadedConvResponse([]string{id}, cursor)))
	}))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{"TweetDetail": "_NvJCnIjOW__EP5-RF197A"})
	result, err := c.GetReplies(context.Background(), "t1", &types.ThreadOptions{
		FetchOptions: types.FetchOptions{MaxPages: 2},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if calls != 2 {
		t.Errorf("want exactly 2 pages (MaxPages=2), got %d calls", calls)
	}
	if result.NextCursor == "" {
		t.Error("want non-empty NextCursor when stopped at MaxPages")
	}
}
