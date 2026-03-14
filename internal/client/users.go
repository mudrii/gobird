package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/mudrii/gobird/internal/types"
)

var (
	htmlUserIDRe = regexp.MustCompile(`(?i)"(?:user_id|userId|account_id|accountId)"\s*:\s*"?(\d+)"?`)
	htmlScreenRe = regexp.MustCompile(`(?i)"(?:screen_name|screenName)"\s*:\s*"([^"]+)"`)
	htmlNameRe   = regexp.MustCompile(`(?i)"name"\s*:\s*"([^"]+)"`)
)

// GetCurrentUser resolves the authenticated user's identity (public wrapper).
func (c *Client) GetCurrentUser(ctx context.Context) (*types.CurrentUserResult, error) {
	return c.getCurrentUser(ctx)
}

// getCurrentUser resolves the authenticated user's identity.
// Tries 4 API endpoints then 2 HTML pages. Does NOT call UserByScreenName.
func (c *Client) getCurrentUser(ctx context.Context) (*types.CurrentUserResult, error) {
	var lastErr error
	apiURLs := []string{
		SettingsURL,
		SettingsAPITwitterURL,
		CredentialsURL,
		CredentialsAPITwitterURL,
	}
	for _, rawURL := range apiURLs {
		u, err := c.tryGetCurrentUserFromAPI(ctx, rawURL)
		if err != nil {
			lastErr = err
			continue
		}
		if u != nil {
			c.userIDMu.Lock()
			if c.userID == "" {
				c.userID = u.ID
			}
			c.userIDMu.Unlock()
			return u, nil
		}
	}

	htmlPages := []string{SettingsPageURL, SettingsPageTwitterURL}
	for _, rawURL := range htmlPages {
		u, err := c.tryGetCurrentUserFromHTML(ctx, rawURL)
		if err != nil {
			lastErr = err
			continue
		}
		if u != nil {
			c.userIDMu.Lock()
			if c.userID == "" {
				c.userID = u.ID
			}
			c.userIDMu.Unlock()
			return u, nil
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("could not resolve current user: %w", lastErr)
	}
	return nil, fmt.Errorf("could not resolve current user: no user ID found in any endpoint response")
}

func (c *Client) tryGetCurrentUserFromAPI(ctx context.Context, rawURL string) (*types.CurrentUserResult, error) {
	body, err := c.doGET(ctx, rawURL, c.getJSONHeaders())
	if err != nil {
		return nil, err
	}

	var v map[string]any
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, err
	}

	id := firstStringLike(
		v["user_id"],
		v["user_id_str"],
		v["id_str"],
		nestedValue(v, "user", "id_str"),
		nestedValue(v, "user", "id"),
		nestedValue(v, "data", "user_id"),
		nestedValue(v, "data", "user_id_str"),
		nestedValue(v, "data", "user", "id_str"),
		nestedValue(v, "data", "user", "id"),
	)
	if id == "" {
		return nil, nil
	}

	return &types.CurrentUserResult{
		ID: id,
		Username: firstStringLike(
			v["screen_name"],
			nestedValue(v, "user", "screen_name"),
			nestedValue(v, "data", "user", "screen_name"),
		),
		Name: firstStringLike(
			v["name"],
			nestedValue(v, "user", "name"),
			nestedValue(v, "data", "user", "name"),
		),
	}, nil
}

func nestedValue(v map[string]any, path ...string) any {
	cur := any(v)
	for _, key := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[key]
	}
	return cur
}

func firstStringLike(values ...any) string {
	for _, v := range values {
		switch t := v.(type) {
		case string:
			if t != "" {
				return t
			}
		case float64:
			if t > 0 {
				return fmt.Sprintf("%.0f", t)
			}
		}
	}
	return ""
}

func (c *Client) tryGetCurrentUserFromHTML(ctx context.Context, rawURL string) (*types.CurrentUserResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("cookie", "auth_token="+c.authToken+"; ct0="+c.ct0)
	req.Header.Set("user-agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	closeErr := resp.Body.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	s := string(body)

	userID := firstCapture(htmlUserIDRe, s)
	if userID == "" {
		return nil, nil
	}
	return &types.CurrentUserResult{
		ID:       userID,
		Username: firstCapture(htmlScreenRe, s),
		Name:     firstCapture(htmlNameRe, s),
	}, nil
}

func firstCapture(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}
