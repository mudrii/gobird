package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	headers, err := c.getJSONHeaders()
	if err != nil {
		return nil, err
	}
	body, err := c.doGET(ctx, rawURL, headers)
	if err != nil {
		return nil, err
	}

	var v map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
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
		case json.Number:
			if s := t.String(); s != "" && s != "0" {
				return s
			}
		}
	}
	return ""
}

func (c *Client) tryGetCurrentUserFromHTML(ctx context.Context, rawURL string) (*types.CurrentUserResult, error) {
	headers := http.Header{}
	headers.Set("cookie", "auth_token="+c.authToken+"; ct0="+c.ct0)
	headers.Set("user-agent", UserAgent)

	body, err := c.doGET(ctx, rawURL, headers)
	if err != nil {
		return nil, err
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
