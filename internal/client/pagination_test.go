package client

import (
	"context"
	"errors"
	"testing"

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
