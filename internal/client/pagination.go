package client

import (
	"context"
	"time"

	"github.com/mudrii/gobird/internal/types"
)

// inlinePageResult is the per-page result from a fetchPage callback used by
// paginateInline. It mirrors the shape returned by each inline-loop fetch.
type inlinePageResult struct {
	tweets     []types.TweetData
	nextCursor string
	success    bool
	err        error
}

// fetchPageFn is the callback type for paginateInline.
type fetchPageFn func(ctx context.Context, cursor string) inlinePageResult

// paginateInline is the shared inline-loop pagination helper used by Search,
// Home, Bookmarks, Likes, BookmarkFolderTimeline, and similar operations.
//
// Stop conditions (Pagination Pattern 2 — inline loops):
//   - nextCursor is empty
//   - nextCursor == cursor (no progress)
//   - page.tweets is empty
//   - added == 0 (all items on page already seen)
//   - maxPages reached
//   - len(accumulated) >= limit (when limit > 0)
//
// Page delay is applied BEFORE the fetch and is skipped for page 0 (correction #9).
// Default delay of 1000 ms is used only when opts.PageDelayMs is 0 and the
// operation documents a delay; callers pass 0 to suppress delay entirely.
func paginateInline(
	ctx context.Context,
	opts types.FetchOptions,
	defaultDelayMs int,
	fetch fetchPageFn,
) types.TweetResult {
	var accumulated []types.TweetData
	cursor := opts.Cursor
	seen := make(map[string]bool)

	delayMs := opts.PageDelayMs
	if delayMs == 0 {
		delayMs = defaultDelayMs
	}

	maxPages := opts.MaxPages
	limit := opts.Limit

	for page := 0; ; page++ {
		// Apply delay before fetch, skip page 0 (correction #9).
		if page > 0 && delayMs > 0 {
			select {
			case <-ctx.Done():
				return types.TweetResult{
					Items:   accumulated,
					Success: len(accumulated) > 0,
					Error:   ctx.Err(),
				}
			case <-time.After(time.Duration(delayMs) * time.Millisecond):
			}
		}

		// Stop if maxPages reached.
		if maxPages > 0 && page >= maxPages {
			break
		}

		result := fetch(ctx, cursor)
		if !result.success {
			if len(accumulated) == 0 {
				return types.TweetResult{
					Success: false,
					Error:   result.err,
				}
			}
			return types.TweetResult{
				Items:      accumulated,
				NextCursor: cursor,
				Success:    false,
				Error:      result.err,
			}
		}

		// Count newly added items (dedup by ID).
		added := 0
		for _, t := range result.tweets {
			if t.ID != "" && !seen[t.ID] {
				seen[t.ID] = true
				accumulated = append(accumulated, t)
				added++
				if limit > 0 && len(accumulated) >= limit {
					break
				}
			}
		}

		nextCursor := result.nextCursor

		// Stop conditions.
		if nextCursor == "" {
			break
		}
		if nextCursor == cursor {
			break
		}
		if len(result.tweets) == 0 {
			break
		}
		if added == 0 {
			break
		}
		if limit > 0 && len(accumulated) >= limit {
			break
		}

		cursor = nextCursor
	}

	return types.TweetResult{
		Items:   accumulated,
		Success: true,
	}
}
