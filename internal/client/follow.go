package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// restFollowError codes (correction #4).
const (
	restErrAlreadyFollowing = 160
	restErrBlocked          = 162
	restErrNotFound         = 108
)

// Follow follows a user by their numeric ID.
// Strategy: REST (x.com/i/api) → REST (api.twitter.com) → GraphQL (correction #4).
func (c *Client) Follow(ctx context.Context, userID string) error {
	// REST-first: try x.com/i/api endpoint.
	if err := c.followViaREST(ctx, FollowRESTURL, userID); err == nil {
		return nil
	} else if isFollowBlocked(err) {
		return err
	} else if isFollowNotFound(err) {
		return err
	}

	// REST second: try api.twitter.com endpoint.
	const apiTwitterFollow = "https://api.twitter.com/1.1/friendships/create.json"
	if err := c.followViaREST(ctx, apiTwitterFollow, userID); err == nil {
		return nil
	} else if isFollowBlocked(err) {
		return err
	} else if isFollowNotFound(err) {
		return err
	}

	// GraphQL fallback.
	return c.followViaGraphQL(ctx, userID)
}

// Unfollow unfollows a user by their numeric ID.
// Same REST-first strategy as Follow.
func (c *Client) Unfollow(ctx context.Context, userID string) error {
	if err := c.followViaREST(ctx, UnfollowRESTURL, userID); err == nil {
		return nil
	} else if isFollowBlocked(err) {
		return err
	} else if isFollowNotFound(err) {
		return err
	}

	const apiTwitterUnfollow = "https://api.twitter.com/1.1/friendships/destroy.json"
	if err := c.followViaREST(ctx, apiTwitterUnfollow, userID); err == nil {
		return nil
	} else if isFollowBlocked(err) {
		return err
	} else if isFollowNotFound(err) {
		return err
	}

	return c.unfollowViaGraphQL(ctx, userID)
}

// followViaREST posts to a REST friendships endpoint (create or destroy).
// Correction #4: Content-Type: application/x-www-form-urlencoded, body: user_id=<id>&skip_status=true.
func (c *Client) followViaREST(ctx context.Context, endpoint, userID string) error {
	params := url.Values{}
	params.Set("user_id", userID)
	params.Set("skip_status", "true")

	headers := c.getBaseHeaders()
	body, err := c.doPOSTForm(ctx, endpoint, headers, params.Encode())
	if err != nil {
		return parseFollowRESTError(err)
	}

	// Treat code 160 (already following) as success.
	var resp map[string]any
	if jsonErr := json.Unmarshal(body, &resp); jsonErr == nil {
		if errs, ok := resp["errors"].([]any); ok {
			for _, e := range errs {
				if em, ok := e.(map[string]any); ok {
					if code, _ := em["code"].(float64); int(code) == restErrAlreadyFollowing {
						return nil
					}
				}
			}
		}
	}
	return nil
}

// followViaGraphQL calls the CreateFriendship GraphQL mutation.
// Correction §var-corrections: body uses snake_case user_id, no features.
func (c *Client) followViaGraphQL(ctx context.Context, userID string) error {
	queryID := c.getQueryID("CreateFriendship")
	body := map[string]any{
		"variables": map[string]any{"user_id": userID},
		"queryId":   queryID,
	}
	headers := c.getJSONHeaders()
	respBody, err := c.doPOSTJSON(ctx, graphqlURL("CreateFriendship", queryID), headers, body)
	if err != nil {
		return err
	}
	return graphQLError(respBody, "CreateFriendship")
}

// unfollowViaGraphQL calls the DestroyFriendship GraphQL mutation.
func (c *Client) unfollowViaGraphQL(ctx context.Context, userID string) error {
	queryID := c.getQueryID("DestroyFriendship")
	body := map[string]any{
		"variables": map[string]any{"user_id": userID},
		"queryId":   queryID,
	}
	headers := c.getJSONHeaders()
	respBody, err := c.doPOSTJSON(ctx, graphqlURL("DestroyFriendship", queryID), headers, body)
	if err != nil {
		return err
	}
	return graphQLError(respBody, "DestroyFriendship")
}

// followError wraps a REST follow/unfollow error code.
type followError struct {
	Code    int
	Message string
}

func (e *followError) Error() string {
	return fmt.Sprintf("follow error %d: %s", e.Code, e.Message)
}

// parseFollowRESTError inspects a REST error response and wraps it if it
// contains a known follow error code.
func parseFollowRESTError(err error) error {
	he, ok := err.(*httpError)
	if !ok {
		return err
	}
	var resp struct {
		Errors []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if jsonErr := json.Unmarshal([]byte(he.Body), &resp); jsonErr != nil {
		return err
	}
	for _, e := range resp.Errors {
		switch e.Code {
		case restErrAlreadyFollowing:
			return nil // treat as success
		case restErrBlocked, restErrNotFound:
			return &followError{Code: e.Code, Message: e.Message}
		}
	}
	return err
}

func isFollowBlocked(err error) bool {
	fe, ok := err.(*followError)
	return ok && fe.Code == restErrBlocked
}

func isFollowNotFound(err error) bool {
	fe, ok := err.(*followError)
	return ok && fe.Code == restErrNotFound
}
