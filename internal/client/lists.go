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

// GetOwnedLists returns the lists owned by the authenticated user.
// Correction #30: variables use isListMembershipShown + isListMemberTargetUserId.
// No cursor even for pagination.
func (c *Client) GetOwnedLists(ctx context.Context, opts *types.FetchOptions) (*types.ListResult, error) {
	if opts == nil {
		opts = &types.FetchOptions{}
	}
	if err := c.ensureClientUserID(ctx); err != nil {
		return nil, err
	}
	return c.fetchLists(ctx, "ListOwnerships", c.cachedUserID(), opts)
}

// GetListMemberships returns the lists the authenticated user is a member of.
// Correction #30: same variable shape as ListOwnerships.
func (c *Client) GetListMemberships(ctx context.Context, opts *types.FetchOptions) (*types.ListResult, error) {
	if opts == nil {
		opts = &types.FetchOptions{}
	}
	if err := c.ensureClientUserID(ctx); err != nil {
		return nil, err
	}
	return c.fetchLists(ctx, "ListMemberships", c.cachedUserID(), opts)
}

// fetchLists paginates a list operation (ListOwnerships or ListMemberships).
func (c *Client) fetchLists(ctx context.Context, operation string, userID string, opts *types.FetchOptions) (*types.ListResult, error) {
	var allLists []types.TwitterList
	seen := map[string]bool{}
	pagesFetched := 0
	cursor := opts.Cursor

	maxPages := opts.MaxPages
	if maxPages <= 0 {
		maxPages = 10
	}

	for {
		if pagesFetched > 0 && opts.PageDelayMs > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(opts.PageDelayMs)*time.Millisecond + paginationJitter()):
			}
		}

		page, err := c.fetchListsPage(ctx, operation, userID, cursor, opts.IncludeRaw)
		if err != nil {
			if len(allLists) > 0 {
				return &types.ListResult{Items: allLists, Success: false, Error: err, NextCursor: cursor}, nil
			}
			return &types.ListResult{Success: false, Error: err}, nil
		}

		pagesFetched++
		for _, l := range page.Items {
			if !seen[l.ID] {
				seen[l.ID] = true
				allLists = append(allLists, l)
			}
		}

		nextCursor := page.NextCursor
		if nextCursor == "" || nextCursor == cursor {
			return &types.ListResult{Items: allLists, Success: true}, nil
		}
		if pagesFetched >= maxPages {
			return &types.ListResult{Items: allLists, Success: true, NextCursor: nextCursor}, nil
		}
		cursor = nextCursor
	}
}

// fetchListsPage fetches a single page of lists.
// Correction #30: ListOwnerships variables:
// {"userId":"<currentUserId>","count":100,"isListMembershipShown":true,"isListMemberTargetUserId":"<currentUserId>"}
// No cursor even for pagination.
func (c *Client) fetchListsPage(ctx context.Context, operation string, userID string, _ string, includeRaw bool) (*types.ListPage, error) {
	queryIDs := c.getQueryIDs(operation)
	features := buildListsFeatures()

	vars := map[string]any{
		"userId":                   userID,
		"count":                    100,
		"isListMembershipShown":    true,
		"isListMemberTargetUserId": userID,
	}
	// Correction #30: no cursor for list pagination.

	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return nil, err
	}
	featuresJSON, err := json.Marshal(features)
	if err != nil {
		return nil, err
	}

	if len(queryIDs) == 0 {
		return nil, fmt.Errorf("%s: no query IDs available", operation)
	}
	var lastErr error
	for _, queryID := range queryIDs {
		reqURL := fmt.Sprintf("%s/%s/%s?variables=%s&features=%s",
			GraphQLBaseURL, queryID, operation,
			url.QueryEscape(string(varsJSON)),
			url.QueryEscape(string(featuresJSON)),
		)
		body, err := c.doGET(ctx, reqURL, c.getJSONHeaders())
		if err != nil {
			lastErr = err
			continue
		}
		return parseListsFromInstructions(body, includeRaw)
	}
	return nil, fmt.Errorf("%s failed: %w", operation, lastErr)
}

// GetListTimeline fetches the timeline for a specific list.
// Correction #45: response path: data.list.tweets_timeline.timeline.instructions.
// Correction #84: variables: {"listId":"<listId>","count":20} + cursor when paginating.
func (c *Client) GetListTimeline(ctx context.Context, listID string, opts *types.FetchOptions) (*types.TweetResult, error) {
	if opts == nil {
		opts = &types.FetchOptions{}
	}

	var allTweets []types.TweetData
	seen := map[string]bool{}
	pagesFetched := 0
	cursor := opts.Cursor

	maxPages := opts.MaxPages
	if maxPages <= 0 {
		maxPages = 10
	}

	for {
		if pagesFetched > 0 && opts.PageDelayMs > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(opts.PageDelayMs)*time.Millisecond + paginationJitter()):
			}
		}

		page, err := c.fetchListTimelinePage(ctx, listID, cursor, opts.QuoteDepth, opts.IncludeRaw)
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

// fetchListTimelinePage fetches a single page of list timeline tweets.
func (c *Client) fetchListTimelinePage(ctx context.Context, listID string, cursor string, quoteDepth int, includeRaw bool) (*types.TweetPage, error) {
	queryIDs := c.getQueryIDs("ListLatestTweetsTimeline")
	features := buildListsFeatures()

	vars := map[string]any{
		"listId": listID,
		"count":  20,
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

	if len(queryIDs) == 0 {
		return nil, fmt.Errorf("ListLatestTweetsTimeline: no query IDs available for list %q", listID)
	}
	var lastErr error
	for _, queryID := range queryIDs {
		reqURL := fmt.Sprintf("%s/%s/ListLatestTweetsTimeline?variables=%s&features=%s",
			GraphQLBaseURL, queryID,
			url.QueryEscape(string(varsJSON)),
			url.QueryEscape(string(featuresJSON)),
		)
		body, err := c.doGET(ctx, reqURL, c.getJSONHeaders())
		if err != nil {
			lastErr = err
			continue
		}
		return parseListTimelineResponse(body, quoteDepth, includeRaw)
	}
	return nil, fmt.Errorf("ListLatestTweetsTimeline failed for list %q: %w", listID, lastErr)
}

// parseListTimelineResponse parses the ListLatestTweetsTimeline response.
// Correction #45: data.list.tweets_timeline.timeline.instructions.
func parseListTimelineResponse(body []byte, quoteDepth int, includeRaw bool) (*types.TweetPage, error) {
	var env struct {
		Data struct {
			List struct {
				TweetsTimeline struct {
					Timeline struct {
						Instructions []types.WireTimelineInstruction `json:"instructions"`
					} `json:"timeline"`
				} `json:"tweets_timeline"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	instructions := env.Data.List.TweetsTimeline.Timeline.Instructions
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

// parseListsFromInstructions parses list items from the GraphQL response.
// Response path: data.user.result.timeline.timeline.instructions.
func parseListsFromInstructions(body []byte, includeRaw bool) (*types.ListPage, error) {
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
	var lists []types.TwitterList
	seen := map[string]bool{}
	// Parse list items from raw JSON since WireItemContent doesn't have list_results.
	var rawEnv struct {
		Data struct {
			User struct {
				Result struct {
					Timeline struct {
						Timeline struct {
							Instructions []json.RawMessage `json:"instructions"`
						} `json:"timeline"`
					} `json:"timeline"`
				} `json:"result"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &rawEnv); err != nil {
		return nil, err
	}
	for _, instRaw := range rawEnv.Data.User.Result.Timeline.Timeline.Instructions {
		var inst struct {
			Entries []struct {
				Content struct {
					ItemContent *struct {
						ListResult *struct {
							Result *types.WireList `json:"result"`
						} `json:"list_results"`
					} `json:"itemContent"`
				} `json:"content"`
			} `json:"entries"`
		}
		if err := json.Unmarshal(instRaw, &inst); err != nil {
			continue
		}
		for _, entry := range inst.Entries {
			if entry.Content.ItemContent == nil || entry.Content.ItemContent.ListResult == nil {
				continue
			}
			wl := entry.Content.ItemContent.ListResult.Result
			if wl == nil || wl.IDStr == "" {
				continue
			}
			if seen[wl.IDStr] {
				continue
			}
			seen[wl.IDStr] = true
			tl := mapWireList(wl)
			lists = append(lists, tl)
		}
	}

	cursor := parsing.ExtractCursorFromInstructions(instructions)
	if includeRaw {
		lists = attachRawToLists(lists, body)
	}
	return &types.ListPage{
		Items:      lists,
		NextCursor: cursor,
		Success:    true,
	}, nil
}

// mapWireList converts a WireList to a TwitterList.
func mapWireList(wl *types.WireList) types.TwitterList {
	tl := types.TwitterList{
		ID:              wl.IDStr,
		Name:            wl.Name,
		Description:     wl.Description,
		MemberCount:     wl.MemberCount,
		SubscriberCount: wl.SubscriberCount,
		IsPrivate:       wl.Mode == "Private",
		CreatedAt:       wl.CreatedAt,
	}
	if wl.UserResults.Result != nil {
		u := parsing.MapUser(wl.UserResults.Result)
		if u != nil {
			tl.Owner = &types.ListOwner{
				ID:       u.ID,
				Username: u.Username,
				Name:     u.Name,
			}
		}
	}
	return tl
}
