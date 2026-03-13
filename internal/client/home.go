package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

// queryUnspecifiedRe matches HomeTimeline's "query: unspecified" refresh trigger.
// Correction #77: case-insensitive, allows optional whitespace between ":" and word.
var queryUnspecifiedRe = regexp.MustCompile(`(?i)query:\s*unspecified`)

// homeTimelinePage fetches a single page from the given home timeline operation
// (either "HomeTimeline" or "HomeLatestTimeline") using the supplied query ID.
func (c *Client) homeTimelinePage(ctx context.Context, operation, queryID, cursor string, count int, quoteDepth int, includeRaw bool) inlinePageResult {
	vars := map[string]any{
		"count":                  count,
		"includePromotedContent": true,
		"latestControlAvailable": true,
		"requestContext":         "launch",
		"withCommunity":          true,
	}
	if cursor != "" {
		vars["cursor"] = cursor
	}

	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return inlinePageResult{success: false, err: err}
	}
	featJSON, err := json.Marshal(buildHomeTimelineFeatures())
	if err != nil {
		return inlinePageResult{success: false, err: err}
	}

	endpoint := graphqlURL(operation, queryID)
	u, err := url.Parse(endpoint)
	if err != nil {
		return inlinePageResult{success: false, err: err}
	}
	q := u.Query()
	q.Set("variables", string(varsJSON))
	q.Set("features", string(featJSON))
	u.RawQuery = q.Encode()

	raw, httpErr := c.doGET(ctx, u.String(), c.getJSONHeaders())
	if httpErr != nil {
		return inlinePageResult{success: false, err: httpErr}
	}

	// Check GraphQL errors — if they match the "query: unspecified" pattern,
	// return as a refresh-triggering failure (correction #77).
	gqlErrs := parseGraphQLErrors(raw)
	for _, e := range gqlErrs {
		if queryUnspecifiedRe.MatchString(e.Message) {
			return inlinePageResult{
				success: false,
				err:     fmt.Errorf("HomeTimeline: %s", e.Message),
			}
		}
	}

	// Navigate response path: data.home.home_timeline_urt.instructions (correction #60).
	var resp struct {
		Data struct {
			Home struct {
				HomeTimelineUrt struct {
					Instructions []types.WireTimelineInstruction `json:"instructions"`
				} `json:"home_timeline_urt"`
			} `json:"home"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return inlinePageResult{success: false, err: err}
	}

	instructions := resp.Data.Home.HomeTimelineUrt.Instructions
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

// isQueryUnspecifiedError reports whether the error message matches the
// "query: unspecified" pattern used by HomeTimeline.
func isQueryUnspecifiedError(err error) bool {
	if err == nil {
		return false
	}
	return queryUnspecifiedRe.MatchString(err.Error())
}

// getHomeTimelineInternal is the shared implementation for GetHomeTimeline and
// GetHomeLatestTimeline. The operation parameter selects which endpoint to use.
// Correction #50: neither returns nextCursor to callers — it is consumed internally.
func (c *Client) getHomeTimelineInternal(ctx context.Context, operation string, opts *types.FetchOptions) types.TweetResult {
	if opts == nil {
		opts = &types.FetchOptions{}
	}
	count := opts.Count
	if count == 0 {
		count = 20
	}

	queryIDs := c.getQueryIDs(operation)
	refreshed := false

	fetchFn := func(ctx context.Context, cursor string) inlinePageResult {
		var lastErr error
		for {
			refreshedThisRound := false
			for _, qid := range queryIDs {
				result := c.homeTimelinePage(ctx, operation, qid, cursor, count, opts.QuoteDepth, opts.IncludeRaw)
				if result.success {
					return result
				}
				lastErr = result.err

				shouldRefresh := is404(lastErr) || isQueryUnspecifiedError(lastErr) ||
					(lastErr != nil && strings.Contains(lastErr.Error(), "query: unspecified"))

				if shouldRefresh && !refreshed {
					refreshed = true
					refreshedThisRound = true
					c.refreshQueryIDs(ctx)
					queryIDs = c.getQueryIDs(operation)
					break
				}
			}
			if !refreshedThisRound {
				return inlinePageResult{success: false, err: lastErr}
			}
		}
	}

	result := paginateInline(ctx, *opts, 0, fetchFn)
	// Correction #50: strip nextCursor before returning to caller.
	result.NextCursor = ""
	return result
}

// GetHomeTimeline fetches the authenticated user's home timeline (algorithmic feed).
// Correction #50: does not return nextCursor to the caller.
// Correction #61: variables include latestControlAvailable and requestContext.
func (c *Client) GetHomeTimeline(ctx context.Context, opts *types.FetchOptions) types.TweetResult {
	return c.getHomeTimelineInternal(ctx, "HomeTimeline", opts)
}

// GetHomeLatestTimeline fetches the authenticated user's chronological home timeline.
// Correction #50: does not return nextCursor to the caller.
func (c *Client) GetHomeLatestTimeline(ctx context.Context, opts *types.FetchOptions) types.TweetResult {
	return c.getHomeTimelineInternal(ctx, "HomeLatestTimeline", opts)
}
