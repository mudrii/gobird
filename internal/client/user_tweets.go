package client

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"time"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

// GetUserTweets fetches all tweets for a user up to opts.Limit, across multiple pages.
// Correction #13, #29: uses UserTweets variables without withV2Timeline.
// Hard max 10 pages.
func (c *Client) GetUserTweets(ctx context.Context, userID string, opts *types.UserTweetsOptions) (*types.TweetResult, error) {
	if opts == nil {
		opts = &types.UserTweetsOptions{}
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	maxPages := opts.MaxPages
	// Hard max 10 pages: hardMaxPages = min(10, maxPages ?? ceil(limit/20))
	calcPages := int(math.Ceil(float64(limit) / 20.0))
	if maxPages <= 0 || maxPages > calcPages {
		maxPages = calcPages
	}
	if maxPages > 10 {
		maxPages = 10
	}

	var allTweets []types.TweetData
	seen := map[string]bool{}
	cursor := opts.Cursor
	pagesFetched := 0

	for {
		// Page delay BEFORE fetches after page 0.
		if pagesFetched > 0 && opts.PageDelayMs > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(opts.PageDelayMs)*time.Millisecond + paginationJitter()):
			}
		}

		page, err := c.fetchUserTweetsPage(ctx, userID, cursor, opts.QuoteDepth, opts.IncludeRaw)
		if err != nil {
			if len(allTweets) > 0 {
				return &types.TweetResult{Items: allTweets, Success: false, Error: err, NextCursor: cursor}, nil
			}
			return &types.TweetResult{Success: false, Error: err}, nil
		}

		pagesFetched++

		for _, t := range page.Items {
			if !seen[t.ID] {
				seen[t.ID] = true
				allTweets = append(allTweets, t)
			}
		}

		nextCursor := page.NextCursor
		if nextCursor == "" || nextCursor == cursor {
			return &types.TweetResult{Items: allTweets, Success: true}, nil
		}
		if pagesFetched >= maxPages {
			return &types.TweetResult{Items: allTweets, Success: true, NextCursor: nextCursor}, nil
		}
		cursor = nextCursor
	}
}

// GetUserTweetsPaged fetches a single page of user tweets, returning the page result.
func (c *Client) GetUserTweetsPaged(ctx context.Context, userID string, cursor string) (*types.TweetPage, error) {
	page, err := c.fetchUserTweetsPage(ctx, userID, cursor, 1, false)
	if err != nil {
		return &types.TweetPage{Success: false, Error: err}, err
	}
	return page, nil
}

// fetchUserTweetsPage fetches a single page of user tweets.
// Correction #13: variables without withV2Timeline.
// Correction #29: fieldToggles: {"withArticlePlainText":false}.
// Response path: data.user.result.timeline.timeline.instructions.
func (c *Client) fetchUserTweetsPage(ctx context.Context, userID string, cursor string, quoteDepth int, includeRaw bool) (*types.TweetPage, error) {
	queryIDs := c.getQueryIDs("UserTweets")
	features := buildUserTweetsFeatures()
	fieldToggles := buildUserTweetsFieldToggles()

	vars := map[string]any{
		"userId":                                 userID,
		"count":                                  20,
		"includePromotedContent":                 false,
		"withQuickPromoteEligibilityTweetFields": true,
		"withVoice":                              true,
	}
	if cursor != "" {
		vars["cursor"] = cursor
	}

	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return nil, err
	}
	featuresJSON, err := json.Marshal(features)
	if err != nil {
		return nil, err
	}
	togglesJSON, err := json.Marshal(fieldToggles)
	if err != nil {
		return nil, err
	}

	if len(queryIDs) == 0 {
		return nil, fmt.Errorf("UserTweets: no query IDs available for user %q", userID)
	}
	var lastErr error
	for _, queryID := range queryIDs {
		reqURL := fmt.Sprintf("%s/%s/UserTweets?variables=%s&features=%s&fieldToggles=%s",
			GraphQLBaseURL, queryID,
			url.QueryEscape(string(varsJSON)),
			url.QueryEscape(string(featuresJSON)),
			url.QueryEscape(string(togglesJSON)),
		)
		body, err := c.doGET(ctx, reqURL, c.getJSONHeaders())
		if err != nil {
			lastErr = err
			continue
		}
		return parseUserTweetsResponse(body, quoteDepth, includeRaw)
	}
	return nil, fmt.Errorf("UserTweets failed for user %q: %w", userID, lastErr)
}

// parseUserTweetsResponse parses the UserTweets response.
// Response path: data.user.result.timeline.timeline.instructions (correction #29).
func parseUserTweetsResponse(body []byte, quoteDepth int, includeRaw bool) (*types.TweetPage, error) {
	var env struct {
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
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	instructions := env.Data.User.Result.Timeline.Timeline.Instructions
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
