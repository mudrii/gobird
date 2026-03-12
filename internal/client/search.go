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

// mustBeDefinedRe matches GraphQL errors whose message says a variable must be defined.
var mustBeDefinedRe = regexp.MustCompile(`(?i)must be defined`)

// isSearchQueryIDMismatch reports whether the given GraphQL error indicates a
// query-ID mismatch for SearchTimeline. Triggers a query ID refresh.
// Correction #76: checks extensions.code == GRAPHQL_VALIDATION_FAILED OR
// (path includes "rawQuery" AND message matches /must be defined/i).
func isSearchQueryIDMismatch(errs []graphqlError) bool {
	for _, e := range errs {
		if e.Extensions.Code == "GRAPHQL_VALIDATION_FAILED" {
			return true
		}
		pathHasRawQuery := false
		for _, p := range e.Path {
			if s, ok := p.(string); ok && s == "rawQuery" {
				pathHasRawQuery = true
				break
			}
		}
		if pathHasRawQuery && mustBeDefinedRe.MatchString(e.Message) {
			return true
		}
	}
	return false
}

// searchPage fetches a single page of search results for the given query, cursor,
// and query ID.
func (c *Client) searchPage(ctx context.Context, queryID, q, cursor string, count int, product string) inlinePageResult {
	vars := map[string]any{
		"rawQuery":    q,
		"count":       count,
		"querySource": "typed_query",
		"product":     product,
	}
	if cursor != "" {
		vars["cursor"] = cursor
	}

	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return inlinePageResult{success: false, err: err}
	}

	endpoint := graphqlURL("SearchTimeline", queryID)
	u, err := url.Parse(endpoint)
	if err != nil {
		return inlinePageResult{success: false, err: err}
	}
	q2 := u.Query()
	q2.Set("variables", string(varsJSON))
	u.RawQuery = q2.Encode()

	body := map[string]any{
		"features": buildSearchFeatures(),
		"queryId":  queryID,
	}

	raw, httpErr := c.doPOSTJSON(ctx, u.String(), c.getJsonHeaders(), body)
	if httpErr != nil {
		return inlinePageResult{success: false, err: httpErr}
	}

	// Parse GraphQL errors from body.
	gqlErrs := parseGraphQLErrors(raw)
	if len(gqlErrs) > 0 {
		if isSearchQueryIDMismatch(gqlErrs) {
			return inlinePageResult{success: false, err: fmt.Errorf("GRAPHQL_VALIDATION_FAILED: %s", gqlErrs[0].Message)}
		}
	}

	// Navigate response path: data.search_by_raw_query.search_timeline.timeline.instructions
	var resp struct {
		Data struct {
			SearchByRawQuery struct {
				SearchTimeline struct {
					Timeline struct {
						Instructions []types.WireTimelineInstruction `json:"instructions"`
					} `json:"timeline"`
				} `json:"search_timeline"`
			} `json:"search_by_raw_query"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return inlinePageResult{success: false, err: err}
	}

	instructions := resp.Data.SearchByRawQuery.SearchTimeline.Timeline.Instructions
	tweets := parsing.ParseTweetsFromInstructions(instructions)
	nextCursor := parsing.ExtractCursorFromInstructions(instructions)

	return inlinePageResult{
		tweets:     tweets,
		nextCursor: nextCursor,
		success:    true,
	}
}

// Search fetches a single page of search results for the given query.
// Returns at most one page; use GetAllSearchResults for full pagination.
// Correction #35: SearchTimeline uses POST with variables in URL query string.
// Correction #76: refreshes on GRAPHQL_VALIDATION_FAILED or HTTP 400/422 with that body.
func (c *Client) Search(ctx context.Context, q string, opts *types.SearchOptions) types.TweetPage {
	if opts == nil {
		opts = &types.SearchOptions{}
	}
	product := opts.Product
	if product == "" {
		product = "Latest"
	}
	count := opts.Count
	if count == 0 {
		count = 20
	}
	cursor := opts.Cursor

	queryIDs := c.getQueryIDs("SearchTimeline")
	var lastErr error
	refreshed := false

	for attempt := 0; attempt < 2; attempt++ {
		for _, qid := range queryIDs {
			result := c.searchPage(ctx, qid, q, cursor, count, product)
			if result.success {
				return types.TweetPage{
					Items:      result.tweets,
					NextCursor: result.nextCursor,
					Success:    true,
				}
			}
			lastErr = result.err

			// Check for HTTP 400/422 with GRAPHQL_VALIDATION_FAILED in body or error.
			shouldRefresh := false
			if he, ok := lastErr.(*httpError); ok {
				if (he.StatusCode == 400 || he.StatusCode == 422) &&
					strings.Contains(he.Body, "GRAPHQL_VALIDATION_FAILED") {
					shouldRefresh = true
				}
			}
			if !shouldRefresh && lastErr != nil &&
				strings.Contains(lastErr.Error(), "GRAPHQL_VALIDATION_FAILED") {
				shouldRefresh = true
			}

			if shouldRefresh && !refreshed {
				refreshed = true
				c.refreshQueryIDs(ctx)
				queryIDs = c.getQueryIDs("SearchTimeline")
				break
			}

			if is404(lastErr) && !refreshed {
				refreshed = true
				c.refreshQueryIDs(ctx)
				queryIDs = c.getQueryIDs("SearchTimeline")
				break
			}
		}
		if attempt > 0 || (!refreshed) {
			break
		}
	}

	return types.TweetPage{
		Success: false,
		Error:   lastErr,
	}
}

// GetAllSearchResults paginates through all pages of search results for query q.
// Uses the inline-loop stop conditions (Pagination Pattern 2).
// Default page delay: 1000 ms (documented for search).
func (c *Client) GetAllSearchResults(ctx context.Context, q string, opts *types.SearchOptions) types.TweetResult {
	if opts == nil {
		opts = &types.SearchOptions{}
	}
	product := opts.Product
	if product == "" {
		product = "Latest"
	}
	count := opts.Count
	if count == 0 {
		count = 20
	}

	queryIDs := c.getQueryIDs("SearchTimeline")
	refreshed := false

	fetchFn := func(ctx context.Context, cursor string) inlinePageResult {
		var lastErr error
		currentIDs := queryIDs

		for _, qid := range currentIDs {
			result := c.searchPage(ctx, qid, q, cursor, count, product)
			if result.success {
				return result
			}
			lastErr = result.err

			shouldRefresh := false
			if he, ok := lastErr.(*httpError); ok {
				if (he.StatusCode == 400 || he.StatusCode == 422) &&
					strings.Contains(he.Body, "GRAPHQL_VALIDATION_FAILED") {
					shouldRefresh = true
				}
			}
			if !shouldRefresh && lastErr != nil &&
				strings.Contains(lastErr.Error(), "GRAPHQL_VALIDATION_FAILED") {
				shouldRefresh = true
			}
			if is404(lastErr) {
				shouldRefresh = true
			}

			if shouldRefresh && !refreshed {
				refreshed = true
				c.refreshQueryIDs(ctx)
				queryIDs = c.getQueryIDs("SearchTimeline")
				currentIDs = queryIDs
			}
		}
		return inlinePageResult{success: false, err: lastErr}
	}

	return paginateInline(ctx, opts.FetchOptions, 1000, fetchFn)
}
