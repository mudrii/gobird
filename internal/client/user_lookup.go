package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
)

// getUserByScreenNameQueryIDs returns the hardcoded-only query IDs for UserByScreenName.
// Correction #5: never uses runtime cache.
func getUserByScreenNameQueryIDs() []string {
	return PerOperationFallbackIDs["UserByScreenName"]
}

// GetUserIDByUsername resolves a Twitter username to its numeric user ID.
func (c *Client) GetUserIDByUsername(ctx context.Context, username string) (string, error) {
	user, err := c.fetchUserByScreenName(ctx, username)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

// GetUserAboutAccount returns profile information for the given username.
// Uses AboutAccountQuery with withRefreshedQueryIDsOn404.
func (c *Client) GetUserAboutAccount(ctx context.Context, username string) (*types.TwitterUser, error) {
	attempt := func() attemptResult {
		queryID := c.getQueryID("AboutAccountQuery")
		vars := map[string]any{
			"screenName": username,
		}
		varsJSON, err := json.Marshal(vars)
		if err != nil {
			return attemptResult{err: err}
		}
		reqURL := fmt.Sprintf("%s/%s/AboutAccountQuery?variables=%s",
			GraphQLBaseURL, queryID, url.QueryEscape(string(varsJSON)))

		body, err := c.doGET(ctx, reqURL, c.getJSONHeaders())
		if err != nil {
			had404 := is404(err)
			return attemptResult{err: err, had404: had404}
		}
		return attemptResult{body: body, success: true}
	}

	ar, _ := c.withRefreshedQueryIDsOn404(ctx, attempt)
	if !ar.success {
		return nil, ar.err
	}

	var env struct {
		Data struct {
			UserResultByScreenName struct {
				Result struct {
					AboutProfile *aboutProfileWire `json:"about_profile"`
				} `json:"result"`
			} `json:"user_result_by_screen_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ar.body, &env); err != nil {
		return nil, err
	}
	ap := env.Data.UserResultByScreenName.Result.AboutProfile
	if ap == nil {
		return nil, fmt.Errorf("about_profile not found for %q", username)
	}
	return &types.TwitterUser{
		ID:              ap.UserID,
		Username:        ap.ScreenName,
		Name:            ap.Name,
		Description:     ap.Description,
		FollowersCount:  ap.FollowersCount,
		FollowingCount:  ap.FriendsCount,
		ProfileImageURL: ap.ProfileImageURLHTTPS,
		CreatedAt:       ap.CreatedAt,
	}, nil
}

// aboutProfileWire is the wire shape for about_profile.
type aboutProfileWire struct {
	UserID               string `json:"user_id"`
	ScreenName           string `json:"screen_name"`
	Name                 string `json:"name"`
	Description          string `json:"description"`
	FollowersCount       int    `json:"followers_count"`
	FriendsCount         int    `json:"friends_count"`
	ProfileImageURLHTTPS string `json:"profile_image_url_https"`
	CreatedAt            string `json:"created_at"`
}

// fetchUserByScreenName resolves a user by screen name using the hardcoded query IDs.
// Correction #5: UserUnavailable → stop immediately.
// Correction #55: REST fallback uses users/show.json (NOT users/lookup.json).
// Correction #63: variables use snake_case screen_name.
// Correction #79: response also checks core.screen_name and core.name.
// Correction #80: fieldToggles: {"withAuxiliaryUserLabels":false}.
func (c *Client) fetchUserByScreenName(ctx context.Context, username string) (*types.TwitterUser, error) {
	queryIDs := getUserByScreenNameQueryIDs()
	features := buildUserByScreenNameFeatures()
	fieldToggles := buildUserByScreenNameFieldToggles()

	vars := map[string]any{
		"screen_name":              username,
		"withSafetyModeUserFields": true,
	}

	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return nil, err
	}
	featuresJSON, err := json.Marshal(features)
	if err != nil {
		return nil, err
	}
	togglesJSON, err := json.Marshal(fieldToggles)
	if err != nil {
		return nil, err
	}

	for _, queryID := range queryIDs {
		reqURL := fmt.Sprintf("%s/%s/UserByScreenName?variables=%s&features=%s&fieldToggles=%s",
			GraphQLBaseURL, queryID,
			url.QueryEscape(string(varsJSON)),
			url.QueryEscape(string(featuresJSON)),
			url.QueryEscape(string(togglesJSON)),
		)
		body, err := c.doGET(ctx, reqURL, c.getJSONHeaders())
		if err != nil {
			if is404(err) {
				continue
			}
			return nil, err
		}
		user, unavailable, parseErr := parseUserByScreenNameResponse(body, username)
		if parseErr != nil {
			continue
		}
		if unavailable {
			return nil, fmt.Errorf("user %q is unavailable", username)
		}
		if user != nil {
			return user, nil
		}
	}

	// REST fallback: users/show.json (correction #55).
	restURL := fmt.Sprintf("%s?screen_name=%s", UserLookupRESTURL, url.QueryEscape(username))
	body, err := c.doGET(ctx, restURL, c.getJSONHeaders())
	if err != nil {
		return nil, fmt.Errorf("user %q not found: %w", username, err)
	}
	return parseRESTUserShowResponse(body)
}

// buildUserByScreenNameFeatures returns the feature set for UserByScreenName.
// Uses buildArticleFeatures as base (same as UserTweets without the custom flags).
func buildUserByScreenNameFeatures() map[string]any {
	return buildArticleFeatures()
}

// parseUserByScreenNameResponse parses the GraphQL response for UserByScreenName.
// Returns (user, isUnavailable, error).
// Correction #79: also checks core.screen_name and core.name.
func parseUserByScreenNameResponse(body []byte, _ string) (*types.TwitterUser, bool, error) {
	var env struct {
		Data struct {
			User *types.WireUserResult `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, false, err
	}
	if env.Data.User == nil || env.Data.User.Result == nil {
		return nil, false, fmt.Errorf("no user result")
	}
	raw := env.Data.User.Result
	if raw.TypeName == "UserUnavailable" {
		return nil, true, nil
	}
	user := parsing.MapUser(raw)
	if user == nil {
		return nil, false, fmt.Errorf("could not map user")
	}
	// Correction #79: check core.screen_name / core.name if legacy fields are empty.
	if user.Username == "" {
		// Try to parse a core block directly from raw JSON.
		var coreEnv struct {
			Data struct {
				User struct {
					Result struct {
						Core struct {
							ScreenName string `json:"screen_name"`
							Name       string `json:"name"`
						} `json:"core"`
					} `json:"result"`
				} `json:"user"`
			} `json:"data"`
		}
		if json.Unmarshal(body, &coreEnv) == nil {
			if coreEnv.Data.User.Result.Core.ScreenName != "" {
				user.Username = coreEnv.Data.User.Result.Core.ScreenName
			}
			if coreEnv.Data.User.Result.Core.Name != "" {
				user.Name = coreEnv.Data.User.Result.Core.Name
			}
		}
	}
	return user, false, nil
}

// parseRESTUserShowResponse parses the REST users/show.json response.
func parseRESTUserShowResponse(body []byte) (*types.TwitterUser, error) {
	var v struct {
		IDStr                string `json:"id_str"`
		ScreenName           string `json:"screen_name"`
		Name                 string `json:"name"`
		Description          string `json:"description"`
		FollowersCount       int    `json:"followers_count"`
		FriendsCount         int    `json:"friends_count"`
		ProfileImageURLHTTPS string `json:"profile_image_url_https"`
		CreatedAt            string `json:"created_at"`
		Verified             bool   `json:"verified"`
	}
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, err
	}
	if v.IDStr == "" {
		return nil, fmt.Errorf("REST user show returned no id_str")
	}
	return &types.TwitterUser{
		ID:              v.IDStr,
		Username:        v.ScreenName,
		Name:            v.Name,
		Description:     v.Description,
		FollowersCount:  v.FollowersCount,
		FollowingCount:  v.FriendsCount,
		ProfileImageURL: v.ProfileImageURLHTTPS,
		CreatedAt:       v.CreatedAt,
	}, nil
}
