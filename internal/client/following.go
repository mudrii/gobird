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

// GetFollowing returns the list of users that userID is following.
// Correction #14, #82: uses withRefreshedQueryIDsOn404; REST fallback only when refreshed=true.
func (c *Client) GetFollowing(ctx context.Context, userID string, opts *types.FetchOptions) (*types.UserResult, error) {
	if opts == nil {
		opts = &types.FetchOptions{}
	}
	return c.paginateFollowOp(ctx, "Following", userID, opts)
}

// GetFollowers returns the list of users following userID.
// Correction #14, #82: uses withRefreshedQueryIDsOn404; REST fallback only when refreshed=true.
func (c *Client) GetFollowers(ctx context.Context, userID string, opts *types.FetchOptions) (*types.UserResult, error) {
	if opts == nil {
		opts = &types.FetchOptions{}
	}
	return c.paginateFollowOp(ctx, "Followers", userID, opts)
}

// paginateFollowOp paginates a following or followers operation.
func (c *Client) paginateFollowOp(ctx context.Context, operation string, userID string, opts *types.FetchOptions) (*types.UserResult, error) {
	var allUsers []types.TwitterUser
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

		page, err := c.fetchFollowPageWithRefresh(ctx, operation, userID, cursor, opts.IncludeRaw)
		if err != nil {
			if len(allUsers) > 0 {
				return &types.UserResult{Items: allUsers, Success: false, Error: err, NextCursor: cursor}, nil
			}
			return &types.UserResult{Success: false, Error: err}, nil
		}

		pagesFetched++
		for _, u := range page.Items {
			if !seen[u.ID] {
				seen[u.ID] = true
				allUsers = append(allUsers, u)
			}
		}

		nextCursor := page.NextCursor
		if nextCursor == "" || nextCursor == cursor {
			return &types.UserResult{Items: allUsers, Success: true}, nil
		}
		if pagesFetched >= maxPages {
			return &types.UserResult{Items: allUsers, Success: true, NextCursor: nextCursor}, nil
		}
		cursor = nextCursor
	}
}

// fetchFollowPageWithRefresh fetches a single page with withRefreshedQueryIDsOn404.
// Correction #82: REST fallback only when refreshed=true after 404.
func (c *Client) fetchFollowPageWithRefresh(ctx context.Context, operation string, userID string, cursor string, includeRaw bool) (*types.UserPage, error) {
	var page *types.UserPage

	attempt := func() attemptResult {
		p, err := c.fetchFollowPage(ctx, operation, userID, cursor, includeRaw)
		if err != nil {
			had404 := is404(err)
			return attemptResult{err: err, had404: had404}
		}
		pageBytes, marshalErr := json.Marshal(p)
		if marshalErr != nil {
			return attemptResult{err: marshalErr}
		}
		page = p
		return attemptResult{body: pageBytes, success: true}
	}

	ar, refreshed := c.withRefreshedQueryIDsOn404(ctx, attempt)
	if ar.success {
		return page, nil
	}

	if refreshed && ar.had404 {
		// REST fallback (only when refreshed=true after 404).
		return c.fetchFollowPageREST(ctx, operation, userID, cursor, includeRaw)
	}

	return nil, ar.err
}

// fetchFollowPage fetches a single page using GraphQL.
// Correction #14: variables: {"userId":"<userId>","count":20,"includePromotedContent":false}.
// Response path: data.user.result.timeline.timeline.instructions.
func (c *Client) fetchFollowPage(ctx context.Context, operation string, userID string, cursor string, includeRaw bool) (*types.UserPage, error) {
	queryIDs := c.getQueryIDs(operation)
	features := buildFollowingFeatures()

	vars := map[string]any{
		"userId":                 userID,
		"count":                  20,
		"includePromotedContent": false,
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
		return parseFollowResponse(body, includeRaw)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("%s: no query IDs available", operation)
}

// fetchFollowPageREST fetches following/followers via REST fallback.
// Correction #82: GET https://x.com/i/api/1.1/friends/list.json for following,
// https://x.com/i/api/1.1/followers/list.json for followers.
// Fallback: api.twitter.com mirror.
func (c *Client) fetchFollowPageREST(ctx context.Context, operation string, userID string, cursor string, includeRaw bool) (*types.UserPage, error) {
	var primaryURL, fallbackURL string
	count := 20

	if operation == "Following" {
		primaryURL = fmt.Sprintf("%s?user_id=%s&count=%d&skip_status=true&include_user_entities=false",
			FollowingRESTURL, url.QueryEscape(userID), count)
		fallbackURL = fmt.Sprintf("https://api.twitter.com/1.1/friends/list.json?user_id=%s&count=%d&skip_status=true&include_user_entities=false",
			url.QueryEscape(userID), count)
	} else {
		primaryURL = fmt.Sprintf("%s?user_id=%s&count=%d&skip_status=true&include_user_entities=false",
			FollowersRESTURL, url.QueryEscape(userID), count)
		fallbackURL = fmt.Sprintf("https://api.twitter.com/1.1/followers/list.json?user_id=%s&count=%d&skip_status=true&include_user_entities=false",
			url.QueryEscape(userID), count)
	}

	if cursor != "" {
		primaryURL += "&cursor=" + url.QueryEscape(cursor)
		fallbackURL += "&cursor=" + url.QueryEscape(cursor)
	}

	body, err := c.doGET(ctx, primaryURL, c.getJSONHeaders())
	if err != nil {
		body, err = c.doGET(ctx, fallbackURL, c.getJSONHeaders())
		if err != nil {
			return nil, err
		}
	}
	return parseRESTFollowResponse(body, includeRaw)
}

// parseFollowResponse parses the GraphQL following/followers response.
// Response path: data.user.result.timeline.timeline.instructions.
func parseFollowResponse(body []byte, includeRaw bool) (*types.UserPage, error) {
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
	users := parsing.ParseUsersFromInstructions(instructions)
	if includeRaw {
		users = attachRawToUsers(users, body)
	}
	cursor := parsing.ExtractCursorFromInstructions(instructions)
	return &types.UserPage{
		Items:      users,
		NextCursor: cursor,
		Success:    true,
	}, nil
}

// parseRESTFollowResponse parses the REST following/followers response.
// REST returns {"users":[...],"next_cursor_str":"..."}.
func parseRESTFollowResponse(body []byte, includeRaw bool) (*types.UserPage, error) {
	var env struct {
		Users []struct {
			IDStr                string `json:"id_str"`
			ScreenName           string `json:"screen_name"`
			Name                 string `json:"name"`
			Description          string `json:"description"`
			FollowersCount       int    `json:"followers_count"`
			FriendsCount         int    `json:"friends_count"`
			ProfileImageURLHTTPS string `json:"profile_image_url_https"`
			CreatedAt            string `json:"created_at"`
			Verified             bool   `json:"verified"`
		} `json:"users"`
		NextCursorStr string `json:"next_cursor_str"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	var users []types.TwitterUser
	for _, u := range env.Users {
		users = append(users, types.TwitterUser{
			ID:              u.IDStr,
			Username:        u.ScreenName,
			Name:            u.Name,
			Description:     u.Description,
			FollowersCount:  u.FollowersCount,
			FollowingCount:  u.FriendsCount,
			IsBlueVerified:  u.Verified,
			ProfileImageURL: u.ProfileImageURLHTTPS,
			CreatedAt:       u.CreatedAt,
		})
	}
	if includeRaw {
		users = attachRawToUsers(users, body)
	}
	nextCursor := env.NextCursorStr
	if nextCursor == "0" {
		nextCursor = ""
	}
	return &types.UserPage{
		Items:      users,
		NextCursor: nextCursor,
		Success:    true,
	}, nil
}
