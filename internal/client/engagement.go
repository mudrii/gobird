package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// Like likes a tweet. Returns nil on success.
func (c *Client) Like(ctx context.Context, tweetID string) error {
	queryID := c.getQueryID("FavoriteTweet")
	body := map[string]any{
		"variables": map[string]any{"tweet_id": tweetID},
		"queryId":   queryID,
	}
	headers := c.getJsonHeaders()
	headers.Set("referer", "https://x.com/i/status/"+tweetID)
	_, err := c.doPOSTJSON(ctx, graphqlURL("FavoriteTweet", queryID), headers, body)
	return err
}

// Unlike unlikes a tweet. Returns nil on success.
func (c *Client) Unlike(ctx context.Context, tweetID string) error {
	queryID := c.getQueryID("UnfavoriteTweet")
	body := map[string]any{
		"variables": map[string]any{"tweet_id": tweetID},
		"queryId":   queryID,
	}
	headers := c.getJsonHeaders()
	headers.Set("referer", "https://x.com/i/status/"+tweetID)
	_, err := c.doPOSTJSON(ctx, graphqlURL("UnfavoriteTweet", queryID), headers, body)
	return err
}

// Retweet retweets a tweet. Returns the retweet ID on success.
func (c *Client) Retweet(ctx context.Context, tweetID string) (string, error) {
	queryID := c.getQueryID("CreateRetweet")
	body := map[string]any{
		"variables": map[string]any{"tweet_id": tweetID},
		"queryId":   queryID,
	}
	headers := c.getJsonHeaders()
	headers.Set("referer", "https://x.com/i/status/"+tweetID)
	respBody, err := c.doPOSTJSON(ctx, graphqlURL("CreateRetweet", queryID), headers, body)
	if err != nil {
		return "", err
	}
	return extractRetweetID(respBody)
}

// extractRetweetID pulls the retweet ID from the CreateRetweet response.
func extractRetweetID(body []byte) (string, error) {
	var resp struct {
		Data struct {
			CreateRetweet struct {
				RetweetResults struct {
					Result struct {
						RestID string `json:"rest_id"`
					} `json:"result"`
				} `json:"retweet_results"`
			} `json:"create_retweet"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse CreateRetweet response: %w", err)
	}
	id := resp.Data.CreateRetweet.RetweetResults.Result.RestID
	if id == "" {
		return "", fmt.Errorf("CreateRetweet: empty rest_id in response")
	}
	return id, nil
}

// Unretweet undoes a retweet. tweetID is the source tweet ID (correction #3).
func (c *Client) Unretweet(ctx context.Context, tweetID string) error {
	queryID := c.getQueryID("DeleteRetweet")
	// Correction #3: DeleteRetweet body uses tweet_id and source_tweet_id (both set to tweetID).
	body := map[string]any{
		"variables": map[string]any{
			"tweet_id":       tweetID,
			"source_tweet_id": tweetID,
		},
		"queryId": queryID,
	}
	headers := c.getJsonHeaders()
	headers.Set("referer", "https://x.com/i/status/"+tweetID)
	_, err := c.doPOSTJSON(ctx, graphqlURL("DeleteRetweet", queryID), headers, body)
	return err
}

// Bookmark adds a tweet to the authenticated user's bookmarks.
func (c *Client) Bookmark(ctx context.Context, tweetID string) error {
	queryID := c.getQueryID("CreateBookmark")
	body := map[string]any{
		"variables": map[string]any{"tweet_id": tweetID},
		"queryId":   queryID,
	}
	headers := c.getJsonHeaders()
	headers.Set("referer", "https://x.com/i/status/"+tweetID)
	_, err := c.doPOSTJSON(ctx, graphqlURL("CreateBookmark", queryID), headers, body)
	return err
}
