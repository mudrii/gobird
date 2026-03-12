package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mudrii/gobird/internal/types"
)

// GetCurrentUser resolves the authenticated user's identity (public wrapper).
func (c *Client) GetCurrentUser(ctx context.Context) (*types.CurrentUserResult, error) {
	return c.getCurrentUser(ctx)
}

// getCurrentUser resolves the authenticated user's identity.
// Tries 4 API endpoints then 2 HTML pages. Does NOT call UserByScreenName.
// Correction #6.
func (c *Client) getCurrentUser(ctx context.Context) (*types.CurrentUserResult, error) {
	apiURLs := []string{
		SettingsURL,
		CredentialsURL,
		"https://x.com/i/api/1.1/account/verify_credentials.json",
		"https://api.twitter.com/1.1/account/verify_credentials.json",
	}
	for _, url := range apiURLs {
		if u := c.tryGetCurrentUserFromAPI(ctx, url); u != nil {
			return u, nil
		}
	}
	htmlPages := []string{SettingsPageURL, "https://x.com/home"}
	for _, url := range htmlPages {
		if u := c.tryGetCurrentUserFromHTML(ctx, url); u != nil {
			return u, nil
		}
	}
	return nil, fmt.Errorf("could not resolve current user")
}

func (c *Client) tryGetCurrentUserFromAPI(ctx context.Context, url string) *types.CurrentUserResult {
	body, err := c.doGET(ctx, url, c.getJsonHeaders())
	if err != nil {
		return nil
	}
	var v map[string]any
	if err := json.Unmarshal(body, &v); err != nil {
		return nil
	}
	// Try data.user.id (only accepted when type is string — correction #78).
	if data, ok := v["data"].(map[string]any); ok {
		if user, ok := data["user"].(map[string]any); ok {
			if id, ok := user["id"].(string); ok && id != "" {
				name, _ := user["name"].(string)
				username, _ := user["screen_name"].(string)
				return &types.CurrentUserResult{ID: id, Username: username, Name: name}
			}
		}
	}
	// Also try top-level id_str / screen_name (verify_credentials shape).
	idStr, _ := v["id_str"].(string)
	screenName, _ := v["screen_name"].(string)
	name, _ := v["name"].(string)
	if idStr != "" {
		return &types.CurrentUserResult{ID: idStr, Username: screenName, Name: name}
	}
	return nil
}

func (c *Client) tryGetCurrentUserFromHTML(_ context.Context, _ string) *types.CurrentUserResult {
	// TODO(Phase 4): implement HTML scraping for current user fallback.
	return nil
}
