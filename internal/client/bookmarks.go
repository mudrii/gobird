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
	respBody, err := c.doPOSTJSON(ctx, graphqlURL("DeleteBookmark", queryID), headers, body)
	if err != nil {
		return err
	}
	return graphQLError(respBody, "DeleteBookmark")
}
