package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

// bookmarkPage fetches a single page of bookmarks using fetchWithRetry.
// Correction #85: variables are built once outside the per-queryId loop (cursor baked in).
// Correction #16: fetchWithRetry with maxRetries=2, baseDelayMs=500.
func (c *Client) bookmarkPage(ctx context.Context, queryID string, varsJSON, featJSON []byte, quoteDepth int, includeRaw bool) inlinePageResult {
	endpoint := graphqlURL("Bookmarks", queryID)
	u, err := url.Parse(endpoint)
	if err != nil {
		return inlinePageResult{success: false, err: err}
	}
	q := u.Query()
	q.Set("variables", string(varsJSON))
	q.Set("features", string(featJSON))
	u.RawQuery = q.Encode()

	raw, httpErr := c.fetchWithRetry(ctx, u.String(), c.getJSONHeaders())
	if httpErr != nil {
		return inlinePageResult{success: false, err: httpErr}
	}

	// Navigate response path: data.bookmark_timeline_v2.timeline.instructions
	var resp struct {
		Data struct {
			BookmarkTimelineV2 struct {
				Timeline struct {
					Instructions []types.WireTimelineInstruction `json:"instructions"`
				} `json:"timeline"`
			} `json:"bookmark_timeline_v2"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return inlinePageResult{success: false, err: err}
	}

	instructions := resp.Data.BookmarkTimelineV2.Timeline.Instructions
	tweets := parsing.ParseTweetsFromInstructionsWithOptions(instructions, parsing.TweetParseOptions{QuoteDepth: quoteDepth})
	if includeRaw {
		tweets = attachRawToTweets(tweets, raw)
	}
	nextCursor := parsing.ExtractCursorFromInstructions(instructions)

	return inlinePageResult{
		tweets:     tweets,
		nextCursor: nextCursor,
		success:    true,
	}
}

// GetBookmarks paginates through all bookmarked tweets for the authenticated user.
// Correction #22: variables include all 6 fields.
// Correction #85: variables built once per page call (outside queryId loop).
// Correction #16: uses fetchWithRetry.
func (c *Client) GetBookmarks(ctx context.Context, opts *types.FetchOptions) types.TweetResult {
	if opts == nil {
		opts = &types.FetchOptions{}
	}
	count := opts.Count
	if count == 0 {
		count = 20
	}

	featJSON, err := json.Marshal(buildBookmarksFeatures())
	if err != nil {
		return types.TweetResult{Success: false, Error: err}
	}

	queryIDs := c.getQueryIDs("Bookmarks")
	refreshed := false

	fetchFn := func(ctx context.Context, cursor string) inlinePageResult {
		// Correction #85: build variables once per page call, shared across queryId retries.
		vars := map[string]any{
			"count":                    count,
			"includePromotedContent":   false,
			"withDownvotePerspective":  false,
			"withReactionsMetadata":    false,
			"withReactionsPerspective": false,
		}
		if cursor != "" {
			vars["cursor"] = cursor
		}
		varsJSON, err := json.Marshal(vars)
		if err != nil {
			return inlinePageResult{success: false, err: err}
		}

		var lastErr error
		for {
			refreshedThisRound := false
			for _, qid := range queryIDs {
				result := c.bookmarkPage(ctx, qid, varsJSON, featJSON, opts.QuoteDepth, opts.IncludeRaw)
				if result.success {
					return result
				}
				lastErr = result.err

				if is404(lastErr) && !refreshed {
					refreshed = true
					refreshedThisRound = true
					c.refreshQueryIDs(ctx)
					queryIDs = c.getQueryIDs("Bookmarks")
					break
				}
			}
			if !refreshedThisRound {
				return inlinePageResult{success: false, err: lastErr}
			}
		}
	}

	return paginateInline(ctx, *opts, 0, fetchFn)
}

// bookmarkFolderPage fetches a single page of a bookmark folder timeline.
// Correction #11: retries without count on "Variable "$count"" error.
// Correction #23: returns error immediately on "Variable "$cursor"" error.
// Correction #75: uses fetchWithRetry.
func (c *Client) bookmarkFolderPage(ctx context.Context, queryID, folderID, cursor string, count int, includeCount bool, quoteDepth int, includeRaw bool) inlinePageResult {
	vars := map[string]any{
		"bookmark_collection_id": folderID,
		"includePromotedContent": true,
	}
	if includeCount {
		vars["count"] = count
	}
	if cursor != "" {
		vars["cursor"] = cursor
	}

	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return inlinePageResult{success: false, err: err}
	}
	featJSON, err := json.Marshal(buildBookmarksFeatures())
	if err != nil {
		return inlinePageResult{success: false, err: err}
	}

	endpoint := graphqlURL("BookmarkFolderTimeline", queryID)
	u, err := url.Parse(endpoint)
	if err != nil {
		return inlinePageResult{success: false, err: err}
	}
	q := u.Query()
	q.Set("variables", string(varsJSON))
	q.Set("features", string(featJSON))
	u.RawQuery = q.Encode()

	raw, httpErr := c.fetchWithRetry(ctx, u.String(), c.getJSONHeaders())
	if httpErr != nil {
		return inlinePageResult{success: false, err: httpErr}
	}

	// Check for variable errors before parsing.
	gqlErrs := parseGraphQLErrors(raw)
	for _, e := range gqlErrs {
		if strings.Contains(e.Message, `Variable "$count"`) {
			// Retry without count (correction #11).
			if includeCount {
				return c.bookmarkFolderPage(ctx, queryID, folderID, cursor, count, false, quoteDepth, includeRaw)
			}
		}
		if strings.Contains(e.Message, `Variable "$cursor"`) && cursor != "" {
			// Return error immediately (correction #23).
			return inlinePageResult{
				success: false,
				err:     fmt.Errorf("bookmark folder pagination rejected the cursor parameter"),
			}
		}
	}

	// Navigate response path: data.bookmark_collection_timeline.timeline.instructions (correction #1)
	var resp struct {
		Data struct {
			BookmarkCollectionTimeline struct {
				Timeline struct {
					Instructions []types.WireTimelineInstruction `json:"instructions"`
				} `json:"timeline"`
			} `json:"bookmark_collection_timeline"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return inlinePageResult{success: false, err: err}
	}

	instructions := resp.Data.BookmarkCollectionTimeline.Timeline.Instructions
	tweets := parsing.ParseTweetsFromInstructionsWithOptions(instructions, parsing.TweetParseOptions{QuoteDepth: quoteDepth})
	if includeRaw {
		tweets = attachRawToTweets(tweets, raw)
	}
	nextCursor := parsing.ExtractCursorFromInstructions(instructions)

	return inlinePageResult{
		tweets:     tweets,
		nextCursor: nextCursor,
		success:    true,
	}
}

// GetBookmarkFolderTimeline paginates through all tweets in a bookmark folder.
// Correction #23: includePromotedContent is true (unlike regular bookmarks).
// Correction #11: retries without count on schema error.
// Correction #75: only one hardcoded fallback query ID.
func (c *Client) GetBookmarkFolderTimeline(ctx context.Context, opts *types.BookmarkFolderOptions) types.TweetResult {
	if opts == nil {
		opts = &types.BookmarkFolderOptions{}
	}
	count := opts.Count
	if count == 0 {
		count = 20
	}
	folderID := opts.FolderID

	queryIDs := c.getQueryIDs("BookmarkFolderTimeline")
	refreshed := false

	fetchFn := func(ctx context.Context, cursor string) inlinePageResult {
		var lastErr error
		for {
			refreshedThisRound := false
			for _, qid := range queryIDs {
				result := c.bookmarkFolderPage(ctx, qid, folderID, cursor, count, true, opts.QuoteDepth, opts.IncludeRaw)
				if result.success {
					return result
				}
				lastErr = result.err

				// Stop immediately on cursor rejection (correction #23).
				if lastErr != nil && strings.Contains(lastErr.Error(), "bookmark folder pagination rejected the cursor parameter") {
					return inlinePageResult{success: false, err: lastErr}
				}

				if is404(lastErr) && !refreshed {
					refreshed = true
					refreshedThisRound = true
					c.refreshQueryIDs(ctx)
					queryIDs = c.getQueryIDs("BookmarkFolderTimeline")
					break
				}
			}
			if !refreshedThisRound {
				return inlinePageResult{success: false, err: lastErr}
			}
		}
	}

	return paginateInline(ctx, opts.FetchOptions, 0, fetchFn)
}

// likesPage fetches a single page of liked tweets for the authenticated user.
func (c *Client) likesPage(ctx context.Context, queryID, userID, cursor string, count int, quoteDepth int, includeRaw bool) inlinePageResult {
	// Correction #29: no withV2Timeline field.
	vars := map[string]any{
		"userId":                 userID,
		"count":                  count,
		"includePromotedContent": false,
		"withClientEventToken":   false,
		"withBirdwatchNotes":     false,
		"withVoice":              true,
	}
	if cursor != "" {
		vars["cursor"] = cursor
	}

	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return inlinePageResult{success: false, err: err}
	}
	featJSON, err := json.Marshal(buildLikesFeatures())
	if err != nil {
		return inlinePageResult{success: false, err: err}
	}

	endpoint := graphqlURL("Likes", queryID)
	u, err := url.Parse(endpoint)
	if err != nil {
		return inlinePageResult{success: false, err: err}
	}
	q := u.Query()
	q.Set("variables", string(varsJSON))
	q.Set("features", string(featJSON))
	u.RawQuery = q.Encode()

	// Likes uses doGET (not fetchWithRetry) — correction #49.
	raw, httpErr := c.doGET(ctx, u.String(), c.getJSONHeaders())
	if httpErr != nil {
		return inlinePageResult{success: false, err: httpErr}
	}

	// Navigate response path: data.user.result.timeline.timeline.instructions
	var resp struct {
		Data struct {
			User struct {
				Result struct {
					Timeline struct {
						Timeline struct {
							Instructions []types.WireTimelineInstruction `json:"instructions"`
						} `json:"timeline"`
					} `json:"timeline"`
				} `json:"result"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return inlinePageResult{success: false, err: err}
	}

	instructions := resp.Data.User.Result.Timeline.Timeline.Instructions
	tweets := parsing.ParseTweetsFromInstructionsWithOptions(instructions, parsing.TweetParseOptions{QuoteDepth: quoteDepth})
	if includeRaw {
		tweets = attachRawToTweets(tweets, raw)
	}
	nextCursor := parsing.ExtractCursorFromInstructions(instructions)

	return inlinePageResult{
		tweets:     tweets,
		nextCursor: nextCursor,
		success:    true,
	}
}

// GetLikes paginates through all liked tweets for the authenticated user.
// Correction #29: variables have no withV2Timeline.
// Correction #49: uses doGET, not fetchWithRetry.
// Correction #51: "Query: Unspecified" (exact case) triggers refresh after all IDs exhausted.
func (c *Client) GetLikes(ctx context.Context, opts *types.FetchOptions) types.TweetResult {
	if opts == nil {
		opts = &types.FetchOptions{}
	}
	count := opts.Count
	if count == 0 {
		count = 20
	}

	if err := c.ensureClientUserID(ctx); err != nil {
		return types.TweetResult{Success: false, Error: err}
	}
	userID := c.userID

	queryIDs := c.getQueryIDs("Likes")
	refreshed := false

	fetchFn := func(ctx context.Context, cursor string) inlinePageResult {
		var lastErr error
		var accumulatedErrMsg string

		for {
			refreshedThisRound := false
			for _, qid := range queryIDs {
				result := c.likesPage(ctx, qid, userID, cursor, count, opts.QuoteDepth, opts.IncludeRaw)
				if result.success {
					return result
				}
				lastErr = result.err
				if lastErr != nil {
					accumulatedErrMsg += lastErr.Error()
				}

				// Correction #51: "Query: Unspecified" (exact case) — continue to next ID.
				if lastErr != nil && strings.Contains(lastErr.Error(), "Query: Unspecified") {
					continue
				}

				if is404(lastErr) && !refreshed {
					refreshed = true
					refreshedThisRound = true
					c.refreshQueryIDs(ctx)
					queryIDs = c.getQueryIDs("Likes")
					break
				}
			}
			if !refreshedThisRound {
				break
			}
		}

		// Correction #51: after exhausting all query IDs, if accumulated error includes
		// "Query: Unspecified" (exact case), refresh and retry once.
		if strings.Contains(accumulatedErrMsg, "Query: Unspecified") && !refreshed {
			refreshed = true
			c.refreshQueryIDs(ctx)
			for _, qid := range c.getQueryIDs("Likes") {
				result := c.likesPage(ctx, qid, userID, cursor, count, opts.QuoteDepth, opts.IncludeRaw)
				if result.success {
					return result
				}
				lastErr = result.err
			}
		}

		return inlinePageResult{success: false, err: lastErr}
	}

	return paginateInline(ctx, *opts, 0, fetchFn)
}
