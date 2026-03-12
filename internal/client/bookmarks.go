package client

import "context"

// Unbookmark removes a tweet from the authenticated user's bookmarks.
func (c *Client) Unbookmark(ctx context.Context, tweetID string) error {
	queryID := c.getQueryID("DeleteBookmark")
	body := map[string]any{
		"variables": map[string]any{"tweet_id": tweetID},
		"queryId":   queryID,
	}
	headers := c.getJsonHeaders()
	headers.Set("referer", "https://x.com/i/status/"+tweetID)
	_, err := c.doPOSTJSON(ctx, graphqlURL("DeleteBookmark", queryID), headers, body)
	return err
}
