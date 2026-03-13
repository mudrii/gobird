package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

// getTweetDetailQueryIDs returns the ordered list of query IDs to try for TweetDetail.
func (c *Client) getTweetDetailQueryIDs() []string {
	return c.getQueryIDs("TweetDetail")
}

// buildTweetDetailVars builds the variables map for TweetDetail.
// Correction #28: no referrer, no count. cursor only when paginating.
func buildTweetDetailVars(focalTweetID string, cursor string) map[string]any {
	vars := map[string]any{
		"focalTweetId":                           focalTweetID,
		"with_rux_injections":                    false,
		"rankingMode":                            "Relevance",
		"includePromotedContent":                 true,
		"withCommunity":                          true,
		"withQuickPromoteEligibilityTweetFields": true,
		"withBirdwatchNotes":                     true,
		"withVoice":                              true,
	}
	if cursor != "" {
		vars["cursor"] = cursor
	}
	return vars
}

// buildFetchTweetDetailFeatures returns features for fetchTweetDetail.
// Correction #28: articles_rest_api_enabled and rweb_video_timestamps_enabled added.
func buildFetchTweetDetailFeatures() map[string]any {
	f := buildTweetDetailFeatures()
	f["articles_rest_api_enabled"] = true
	f["rweb_video_timestamps_enabled"] = true
	return f
}

// fetchTweetDetail performs the GET→POST pattern for TweetDetail.
// Correction #36: for each queryId: GET, if 404 → POST, if 404 → had404=true.
// After loop: if had404, refreshQueryIDs then retry once.
func (c *Client) fetchTweetDetail(ctx context.Context, focalTweetID string, cursor string) ([]byte, error) {
	vars := buildTweetDetailVars(focalTweetID, cursor)
	features := buildFetchTweetDetailFeatures()
	fieldToggles := buildArticleFieldToggles()

	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return nil, fmt.Errorf("fetchTweetDetail: marshal vars: %w", err)
	}
	featuresJSON, err := json.Marshal(features)
	if err != nil {
		return nil, fmt.Errorf("fetchTweetDetail: marshal features: %w", err)
	}
	togglesJSON, err := json.Marshal(fieldToggles)
	if err != nil {
		return nil, fmt.Errorf("fetchTweetDetail: marshal fieldToggles: %w", err)
	}

	tryQueryIDs := func(queryIDs []string) ([]byte, error, bool) {
		had404 := false
		var lastErr error
		for _, queryID := range queryIDs {
			getURL := fmt.Sprintf("%s/%s/TweetDetail?variables=%s&features=%s&fieldToggles=%s",
				GraphQLBaseURL, queryID,
				url.QueryEscape(string(varsJSON)),
				url.QueryEscape(string(featuresJSON)),
				url.QueryEscape(string(togglesJSON)),
			)
			body, err := c.doGET(ctx, getURL, c.getJSONHeaders())
			if err == nil {
				return body, nil, false
			}
			if !is404(err) {
				lastErr = err
				continue
			}
			had404 = true
			// GET returned 404 → try POST.
			postURL := fmt.Sprintf("%s/%s/TweetDetail", GraphQLBaseURL, queryID)
			postBody := map[string]any{
				"variables": vars,
				"features":  features,
				"queryId":   queryID,
			}
			body, postErr := c.doPOSTJSON(ctx, postURL, c.getJSONHeaders(), postBody)
			if postErr == nil {
				return body, nil, false
			}
			lastErr = postErr
			if is404(postErr) {
				continue
			}
		}
		return nil, lastErr, had404
	}

	queryIDs := c.getTweetDetailQueryIDs()
	body, lastErr, had404 := tryQueryIDs(queryIDs)
	if body != nil {
		return body, nil
	}
	if had404 {
		c.refreshQueryIDs(ctx)
		body, lastErr, _ = tryQueryIDs(c.getTweetDetailQueryIDs())
		if body != nil {
			return body, nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("TweetDetail failed for tweet %q", focalTweetID)
}

// GetTweet returns the single tweet for the given ID.
// Response path: data.tweetResult.result (camelCase, correction #58).
func (c *Client) GetTweet(ctx context.Context, tweetID string, opts *types.TweetDetailOptions) (*types.TweetData, error) {
	if opts == nil {
		opts = &types.TweetDetailOptions{}
	}
	body, err := c.fetchTweetDetail(ctx, tweetID, "")
	if err != nil {
		return nil, err
	}
	var env struct {
		Data struct {
			TweetResult *types.WireTweetResult `json:"tweetResult"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	if env.Data.TweetResult == nil || env.Data.TweetResult.Result == nil {
		return nil, fmt.Errorf("tweet %q not found", tweetID)
	}
	raw := parsing.UnwrapTweetResult(env.Data.TweetResult.Result)
	td := parsing.MapTweetResultWithOptions(raw, parsing.TweetParseOptions{QuoteDepth: opts.QuoteDepth})
	if td == nil {
		return nil, fmt.Errorf("could not map tweet %q", tweetID)
	}
	if opts.IncludeRaw {
		var rawAny any
		if err := json.Unmarshal(body, &rawAny); err == nil {
			td.Raw = rawAny
		}
	}
	return td, nil
}

// GetReplies returns the replies for the given tweet, paginated.
// Response path: data.threaded_conversation_with_injections_v2.instructions.
func (c *Client) GetReplies(ctx context.Context, tweetID string, opts *types.ThreadOptions) (*types.TweetResult, error) {
	if opts == nil {
		opts = &types.ThreadOptions{}
	}
	return c.paginateCursor(ctx, tweetID, opts, false)
}

// GetThread fetches the full thread for a tweet, applying filter mode.
// Response path: data.threaded_conversation_with_injections_v2.instructions.
func (c *Client) GetThread(ctx context.Context, tweetID string, opts *types.ThreadOptions) ([]types.TweetWithMeta, error) {
	if opts == nil {
		opts = &types.ThreadOptions{}
	}
	result, err := c.paginateCursor(ctx, tweetID, opts, true)
	if err != nil {
		return nil, err
	}
	tweets := result.Items

	// Apply filter mode.
	var authorID string
	if len(tweets) > 0 {
		authorID = tweets[0].AuthorID
	}
	switch opts.FilterMode {
	case "author_only":
		tweets = parsing.FilterAuthorOnly(tweets, authorID)
	case "full_chain":
		tweets = parsing.FilterFullChain(tweets)
	default: // "author_chain" or empty
		tweets = parsing.FilterAuthorChain(tweets, authorID)
	}

	return parsing.AddThreadMetadata(tweets, authorID), nil
}

// paginateCursor implements the pagination loop for replies/thread.
// Correction: loop terminates on empty cursor OR unchanged cursor ONLY.
// Zero items does NOT stop.
func (c *Client) paginateCursor(ctx context.Context, tweetID string, opts *types.ThreadOptions, isThread bool) (*types.TweetResult, error) {
	_ = isThread
	maxPages := opts.MaxPages
	pageDelayMs := opts.PageDelayMs
	cursor := opts.Cursor

	var allTweets []types.TweetData
	seen := map[string]bool{}
	pagesFetched := 0

	for {
		// Step 1: delay before fetches after page 0.
		if pagesFetched > 0 && pageDelayMs > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(pageDelayMs) * time.Millisecond):
			}
		}

		// Step 2: fetch page.
		page, err := c.fetchThreadPage(ctx, tweetID, cursor, opts.QuoteDepth, opts.IncludeRaw)

		// Step 3: handle failure.
		if err != nil {
			if len(allTweets) > 0 {
				return &types.TweetResult{
					Success:    false,
					Error:      err,
					Items:      allTweets,
					NextCursor: cursor,
				}, nil
			}
			return &types.TweetResult{Success: false, Error: err}, err
		}

		// Step 4: count page.
		pagesFetched++

		// Step 5: deduplicate by ID.
		for _, t := range page.Items {
			if !seen[t.ID] {
				seen[t.ID] = true
				allTweets = append(allTweets, t)
			}
		}

		// Step 6: get page cursor.
		pageCursor := page.NextCursor

		// Step 7: empty or unchanged cursor → done.
		if pageCursor == "" || pageCursor == cursor {
			return &types.TweetResult{Items: allTweets, Success: true}, nil
		}

		// Step 8: max pages reached.
		if maxPages > 0 && pagesFetched >= maxPages {
			return &types.TweetResult{Items: allTweets, Success: true, NextCursor: pageCursor}, nil
		}

		// Step 9: advance cursor.
		cursor = pageCursor
	}
}

// fetchThreadPage fetches a single page from TweetDetail for thread/replies.
func (c *Client) fetchThreadPage(ctx context.Context, tweetID string, cursor string, quoteDepth int, includeRaw bool) (*types.TweetPage, error) {
	body, err := c.fetchTweetDetail(ctx, tweetID, cursor)
	if err != nil {
		return nil, err
	}
	return parseThreadedConversationResponse(body, quoteDepth, includeRaw)
}

// parseThreadedConversationResponse parses the threaded_conversation response.
func parseThreadedConversationResponse(body []byte, quoteDepth int, includeRaw bool) (*types.TweetPage, error) {
	var env struct {
		Data struct {
			ThreadedConversationWithInjectionsV2 struct {
				Instructions []types.WireTimelineInstruction `json:"instructions"`
			} `json:"threaded_conversation_with_injections_v2"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	instructions := env.Data.ThreadedConversationWithInjectionsV2.Instructions
	tweets := parsing.ParseTweetsFromInstructionsWithOptions(instructions, parsing.TweetParseOptions{QuoteDepth: quoteDepth})
	if includeRaw {
		tweets = attachRawToTweets(tweets, body)
	}
	cursor := parsing.ExtractCursorFromInstructions(instructions)
	return &types.TweetPage{
		Items:      tweets,
		NextCursor: cursor,
		Success:    true,
	}, nil
}
