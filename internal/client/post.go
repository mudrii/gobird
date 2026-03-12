package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Tweet posts a new tweet with the given text and returns the tweet ID.
func (c *Client) Tweet(ctx context.Context, text string) (string, error) {
	return c.createTweet(ctx, text, "", nil)
}

// Reply posts a reply to the given tweet ID and returns the new tweet ID.
func (c *Client) Reply(ctx context.Context, text, inReplyToID string) (string, error) {
	return c.createTweet(ctx, text, inReplyToID, nil)
}

// TweetWithMedia posts a new tweet with uploaded media IDs attached.
func (c *Client) TweetWithMedia(ctx context.Context, text string, mediaIDs []string) (string, error) {
	return c.createTweet(ctx, text, "", mediaIDs)
}

// ReplyWithMedia posts a reply with uploaded media IDs attached.
func (c *Client) ReplyWithMedia(ctx context.Context, text, inReplyToID string, mediaIDs []string) (string, error) {
	return c.createTweet(ctx, text, inReplyToID, mediaIDs)
}

// createTweet implements the CreateTweet fallback chain (corrections #39, #72).
func (c *Client) createTweet(ctx context.Context, text, inReplyToID string, mediaIDs []string) (string, error) {
	queryID := c.getQueryID("CreateTweet")
	body := buildCreateTweetBody(text, inReplyToID, mediaIDs, queryID)
	headers := c.getJsonHeaders()
	headers.Set("referer", "https://x.com/compose/post")

	// Attempt 1: POST /graphql/<queryId>/CreateTweet
	respBody, err := c.doPOSTJSON(ctx, graphqlURL("CreateTweet", queryID), headers, body)
	if err != nil && is404(err) {
		// Attempt 2: refresh query IDs, retry with new ID
		c.refreshQueryIDs(ctx)
		queryID = c.getQueryID("CreateTweet")
		body = buildCreateTweetBody(text, inReplyToID, mediaIDs, queryID)

		respBody, err = c.doPOSTJSON(ctx, graphqlURL("CreateTweet", queryID), headers, body)
		if err != nil && is404(err) {
			// Attempt 3: POST https://x.com/i/api/graphql (same body with queryId)
			respBody, err = c.doPOSTJSON(ctx, GraphQLBaseURL, headers, body)
		}
	}

	// Check for error code 226 in any response that came back
	if err == nil && respBody != nil {
		errs := parseGraphQLErrors(respBody)
		for _, e := range errs {
			if e.Extensions.Code == "226" {
				return c.tryStatusUpdateFallback(ctx, text, inReplyToID)
			}
		}
		if gqlErr := graphQLError(respBody, "CreateTweet"); gqlErr != nil {
			return "", gqlErr
		}
		return extractCreateTweetID(respBody)
	}

	if err != nil {
		return "", err
	}
	return extractCreateTweetID(respBody)
}

// buildCreateTweetBody constructs the CreateTweet request body.
func buildCreateTweetBody(text, inReplyToID string, mediaIDs []string, queryID string) map[string]any {
	var mediaEntities []map[string]string
	for _, mediaID := range mediaIDs {
		if mediaID != "" {
			mediaEntities = append(mediaEntities, map[string]string{"media_id": mediaID})
		}
	}
	vars := map[string]any{
		"tweet_text":              text,
		"dark_request":            false,
		"media":                   map[string]any{"media_entities": mediaEntities, "possibly_sensitive": false},
		"semantic_annotation_ids": []any{},
	}
	if inReplyToID != "" {
		vars["reply"] = map[string]any{
			"in_reply_to_tweet_id":   inReplyToID,
			"exclude_reply_user_ids": []any{},
		}
	}
	return map[string]any{
		"variables": vars,
		"features":  buildTweetCreateFeatures(),
		"queryId":   queryID,
	}
}

// extractCreateTweetID extracts the tweet ID from the CreateTweet response.
// Response path: data.create_tweet.tweet_results.result.rest_id (correction #38).
func extractCreateTweetID(body []byte) (string, error) {
	var resp struct {
		Data struct {
			CreateTweet struct {
				TweetResults struct {
					Result struct {
						RestID string `json:"rest_id"`
					} `json:"result"`
				} `json:"tweet_results"`
			} `json:"create_tweet"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse CreateTweet response: %w", err)
	}
	id := resp.Data.CreateTweet.TweetResults.Result.RestID
	if id == "" {
		return "", fmt.Errorf("CreateTweet: empty rest_id in response")
	}
	return id, nil
}

// tryStatusUpdateFallback posts via the v1.1 statuses/update.json endpoint (corrections #39, #40).
func (c *Client) tryStatusUpdateFallback(ctx context.Context, text, inReplyToID string) (string, error) {
	params := url.Values{}
	params.Set("status", text)
	if inReplyToID != "" {
		params.Set("in_reply_to_status_id", inReplyToID)
		params.Set("auto_populate_reply_metadata", "true")
	}

	headers := c.getBaseHeaders()
	body, err := c.doPOSTForm(ctx, StatusUpdateURL, headers, params.Encode())
	if err != nil {
		return "", err
	}

	// Correction #40: prefer id_str, fallback to String(id).
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse statuses/update response: %w", err)
	}
	if idStr, ok := resp["id_str"].(string); ok && idStr != "" {
		return idStr, nil
	}
	if idNum, ok := resp["id"].(float64); ok {
		return strconv.FormatInt(int64(idNum), 10), nil
	}
	return "", fmt.Errorf("statuses/update: could not extract tweet ID")
}
