//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mudrii/gobird/internal/client"
	"github.com/mudrii/gobird/internal/config"
	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/testutil"
	"github.com/mudrii/gobird/internal/types"
)

func fixturesDir() string {
	return filepath.Join("..", "fixtures")
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixturesDir(), name))
	if err != nil {
		t.Fatalf("load fixture %s: %v", name, err)
	}
	return data
}

func newTestClient(t *testing.T, handler http.Handler) *client.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	httpClient := testutil.NewHTTPClientForServer(srv)
	c := client.New("fake-auth-token-value", "fake-ct0-value", &client.Options{
		HTTPClient: httpClient,
		QueryIDCache: map[string]string{
			"TweetDetail":    "testQID",
			"SearchTimeline": "testQID",
			"HomeTimeline":   "testQID",
		},
	})
	return c
}

// --- Auth -> Client -> Parsing pipeline ---

func TestPipeline_ClientGetTweet_ParsesCorrectly(t *testing.T) {
	fixture := loadFixture(t, "tweet_detail_response.json")
	c := newTestClient(t, testutil.StaticHandler(200, string(fixture)))

	tweet, err := c.GetTweet(context.Background(), "3001", nil)
	if err != nil {
		t.Fatalf("GetTweet: %v", err)
	}
	if tweet.ID != "3001" {
		t.Errorf("ID: want 3001, got %q", tweet.ID)
	}
	if tweet.Text != "This is a detailed tweet with some content for testing." {
		t.Errorf("Text: got %q", tweet.Text)
	}
	if tweet.Author.Username != "tweetauthor" {
		t.Errorf("Author.Username: want tweetauthor, got %q", tweet.Author.Username)
	}
	if tweet.Author.Name != "Tweet Author" {
		t.Errorf("Author.Name: want 'Tweet Author', got %q", tweet.Author.Name)
	}
	if !tweet.IsBlueVerified {
		t.Error("IsBlueVerified: want true")
	}
	if tweet.ReplyCount != 12 {
		t.Errorf("ReplyCount: want 12, got %d", tweet.ReplyCount)
	}
	if tweet.RetweetCount != 45 {
		t.Errorf("RetweetCount: want 45, got %d", tweet.RetweetCount)
	}
	if tweet.LikeCount != 200 {
		t.Errorf("LikeCount: want 200, got %d", tweet.LikeCount)
	}
}

func TestPipeline_ClientGetTweet_IncludeRaw(t *testing.T) {
	fixture := loadFixture(t, "tweet_detail_response.json")
	c := newTestClient(t, testutil.StaticHandler(200, string(fixture)))

	tweet, err := c.GetTweet(context.Background(), "3001", &types.TweetDetailOptions{IncludeRaw: true})
	if err != nil {
		t.Fatalf("GetTweet with IncludeRaw: %v", err)
	}
	if tweet.Raw == nil {
		t.Error("Raw should be non-nil when IncludeRaw is true")
	}
}

func TestPipeline_ClientGetTweet_NotFound(t *testing.T) {
	c := newTestClient(t, testutil.StaticHandler(200, `{"data":{"tweetResult":null}}`))

	_, err := c.GetTweet(context.Background(), "0000", nil)
	if err == nil {
		t.Fatal("expected error for null tweetResult")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

// --- Client -> Output formatting pipeline ---

func TestPipeline_TweetToFormattedOutput(t *testing.T) {
	fixture := loadFixture(t, "tweet_detail_response.json")
	c := newTestClient(t, testutil.StaticHandler(200, string(fixture)))

	tweet, err := c.GetTweet(context.Background(), "3001", nil)
	if err != nil {
		t.Fatalf("GetTweet: %v", err)
	}

	formatted := output.FormatTweet(*tweet, output.FormatOptions{
		NoColor: true,
		NoEmoji: true,
	})

	if !strings.Contains(formatted, "@tweetauthor") {
		t.Errorf("formatted output missing handle: %q", formatted)
	}
	if !strings.Contains(formatted, "This is a detailed tweet") {
		t.Errorf("formatted output missing text: %q", formatted)
	}
	if !strings.Contains(formatted, "likes: 200") {
		t.Errorf("formatted output missing like count: %q", formatted)
	}
	if !strings.Contains(formatted, "rts: 45") {
		t.Errorf("formatted output missing retweet count: %q", formatted)
	}
	if !strings.Contains(formatted, "replies: 12") {
		t.Errorf("formatted output missing reply count: %q", formatted)
	}
}

func TestPipeline_TweetToJSON(t *testing.T) {
	fixture := loadFixture(t, "tweet_detail_response.json")
	c := newTestClient(t, testutil.StaticHandler(200, string(fixture)))

	tweet, err := c.GetTweet(context.Background(), "3001", nil)
	if err != nil {
		t.Fatalf("GetTweet: %v", err)
	}

	jsonBytes, err := output.ToJSON(tweet)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	var decoded types.TweetData
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.ID != "3001" {
		t.Errorf("JSON roundtrip ID: want 3001, got %q", decoded.ID)
	}
	if decoded.Author.Username != "tweetauthor" {
		t.Errorf("JSON roundtrip Author.Username: want tweetauthor, got %q", decoded.Author.Username)
	}
}

func TestPipeline_TweetToPlainVsDefault(t *testing.T) {
	tweet := types.TweetData{
		ID:   "42",
		Text: "Test formatting",
		Author: types.TweetAuthor{
			Username: "testuser",
			Name:     "Test User",
		},
		LikeCount:    10,
		RetweetCount: 5,
		ReplyCount:   3,
	}

	plain := output.FormatTweet(tweet, output.FormatOptions{Plain: true})
	rich := output.FormatTweet(tweet, output.FormatOptions{})

	if strings.Contains(plain, "\x1b[") {
		t.Errorf("plain output should not contain ANSI codes: %q", plain)
	}
	if !strings.Contains(rich, "\x1b[") {
		t.Errorf("rich output should contain ANSI codes: %q", rich)
	}

	if strings.Contains(plain, "\xf0") {
		t.Errorf("plain output should not contain emoji bytes: %q", plain)
	}
}

// --- Search pagination across multiple pages ---

func TestPipeline_SearchPagination(t *testing.T) {
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		switch calls {
		case 1:
			w.Write(buildSearchResponse([]string{"p1-1", "p1-2"}, "cursor-page2"))
		case 2:
			w.Write(buildSearchResponse([]string{"p2-1"}, ""))
		default:
			w.Write(buildSearchResponse(nil, ""))
		}
	})

	c := newTestClient(t, handler)
	result := c.GetAllSearchResults(context.Background(), "test", &types.SearchOptions{
		FetchOptions: types.FetchOptions{PageDelayMs: 0},
	})
	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items across 2 pages, got %d", len(result.Items))
	}
	if calls != 2 {
		t.Errorf("expected 2 API calls, got %d", calls)
	}
}

func TestPipeline_SearchPagination_MaxPages(t *testing.T) {
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.Write(buildSearchResponse([]string{"t" + string(rune('0'+calls))}, "cursor-next"))
	})

	c := newTestClient(t, handler)
	result := c.GetAllSearchResults(context.Background(), "test", &types.SearchOptions{
		FetchOptions: types.FetchOptions{MaxPages: 1, PageDelayMs: 0},
	})
	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}
	if calls != 1 {
		t.Errorf("expected exactly 1 API call with MaxPages=1, got %d", calls)
	}
}

func TestPipeline_SearchPagination_Limit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(buildSearchResponse([]string{"a", "b", "c", "d", "e"}, "cursor-next"))
	})

	c := newTestClient(t, handler)
	result := c.GetAllSearchResults(context.Background(), "test", &types.SearchOptions{
		FetchOptions: types.FetchOptions{Limit: 3, PageDelayMs: 0},
	})
	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items with Limit=3, got %d", len(result.Items))
	}
}

// --- Config loading -> Client initialization ---

func TestPipeline_ConfigLoading(t *testing.T) {
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0", "BIRD_CONFIG"} {
		t.Setenv(e, "")
	}

	cfg, err := config.Load(filepath.Join(fixturesDir(), "config_valid.json5"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AuthToken != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Errorf("AuthToken: got %q", cfg.AuthToken)
	}
	if cfg.Ct0 != "abcdefghijklmnopqrstuvwxyz012345" {
		t.Errorf("Ct0: got %q", cfg.Ct0)
	}
	if cfg.TimeoutMs != 5000 {
		t.Errorf("TimeoutMs: want 5000, got %d", cfg.TimeoutMs)
	}
	if cfg.QuoteDepth == nil || *cfg.QuoteDepth != 2 {
		t.Errorf("QuoteDepth: want 2, got %v", cfg.QuoteDepth)
	}

	c := client.New(cfg.AuthToken, cfg.Ct0, &client.Options{
		TimeoutMs: cfg.TimeoutMs,
	})
	if c == nil {
		t.Fatal("client.New returned nil")
	}
}

func TestPipeline_ConfigLoading_InvalidFile(t *testing.T) {
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0", "BIRD_CONFIG"} {
		t.Setenv(e, "")
	}

	_, err := config.Load(filepath.Join(fixturesDir(), "config_invalid.json5"))
	if err == nil {
		t.Fatal("expected error for invalid config file")
	}
}

func TestPipeline_ConfigLoading_MinimalDefaults(t *testing.T) {
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0", "BIRD_CONFIG",
		"BIRD_TIMEOUT_MS", "BIRD_COOKIE_TIMEOUT_MS", "BIRD_QUOTE_DEPTH"} {
		t.Setenv(e, "")
	}

	cfg, err := config.Load(filepath.Join(fixturesDir(), "config_minimal.json5"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.QuoteDepth == nil || *cfg.QuoteDepth != 1 {
		t.Errorf("QuoteDepth default: want 1, got %v", cfg.QuoteDepth)
	}
}

// --- Error pipeline tests ---

func TestPipeline_HTTPError_AuthFailure(t *testing.T) {
	fixture := loadFixture(t, "error_auth.json")
	c := newTestClient(t, testutil.StaticHandler(401, string(fixture)))

	_, err := c.GetTweet(context.Background(), "1", nil)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should contain status code: %v", err)
	}
}

func TestPipeline_HTTPError_RateLimit(t *testing.T) {
	fixture := loadFixture(t, "error_rate_limit.json")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(429)
		w.Write(fixture)
	})
	c := newTestClient(t, handler)

	_, err := c.GetTweet(context.Background(), "1", nil)
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error should contain 429: %v", err)
	}
}

func TestPipeline_HTTPError_ServerError(t *testing.T) {
	c := newTestClient(t, testutil.StaticHandler(500, `{"error":"internal server error"}`))

	_, err := c.GetTweet(context.Background(), "1", nil)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

// --- User formatting pipeline ---

func TestPipeline_UserFormatting(t *testing.T) {
	user := types.TwitterUser{
		ID:              "u500",
		Username:        "tweetauthor",
		Name:            "Tweet Author",
		Description:     "A test user",
		FollowersCount:  15000,
		FollowingCount:  500,
		IsBlueVerified:  true,
		ProfileImageURL: "https://example.com/photo.jpg",
	}

	formatted := output.FormatUser(user, output.FormatOptions{NoColor: true, NoEmoji: true})
	if !strings.Contains(formatted, "@tweetauthor") {
		t.Errorf("missing handle: %q", formatted)
	}
	if !strings.Contains(formatted, "15.0K") {
		t.Errorf("missing formatted follower count: %q", formatted)
	}
	if !strings.Contains(formatted, "[verified]") {
		t.Errorf("missing verified badge: %q", formatted)
	}
}

// --- Search results formatting pipeline ---

func TestPipeline_SearchResultsToOutput(t *testing.T) {
	fixture := loadFixture(t, "search_response.json")
	c := newTestClient(t, testutil.StaticHandler(200, string(fixture)))

	result := c.Search(context.Background(), "test", nil)
	if !result.Success {
		t.Fatalf("Search failed: %v", result.Error)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 search results, got %d", len(result.Items))
	}

	for _, item := range result.Items {
		formatted := output.FormatTweet(item, output.FormatOptions{NoColor: true, NoEmoji: true})
		if formatted == "" {
			t.Error("formatted output should not be empty")
		}
		if !strings.Contains(formatted, "@") {
			t.Errorf("formatted output missing handle: %q", formatted)
		}
	}

	jsonBytes, err := output.ToJSON(result.Items)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	var decoded []types.TweetData
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(decoded) != 2 {
		t.Errorf("JSON roundtrip: expected 2, got %d", len(decoded))
	}
}

// --- helpers ---

func buildSearchResponse(ids []string, nextCursor string) []byte {
	var entries []any
	for _, id := range ids {
		entries = append(entries, map[string]any{
			"entryId": "tweet-" + id,
			"content": map[string]any{
				"itemContent": map[string]any{
					"tweet_results": map[string]any{
						"result": map[string]any{
							"__typename": "Tweet",
							"rest_id":    id,
							"legacy":     map[string]any{"full_text": "text " + id},
							"core": map[string]any{
								"user_results": map[string]any{
									"result": map[string]any{
										"__typename": "User",
										"rest_id":    "u1",
										"legacy":     map[string]any{"screen_name": "user", "name": "User"},
									},
								},
							},
						},
					},
				},
			},
		})
	}
	if nextCursor != "" {
		entries = append(entries, map[string]any{
			"entryId": "cursor-bottom",
			"content": map[string]any{
				"cursorType": "Bottom",
				"value":      nextCursor,
			},
		})
	}
	body, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"search_by_raw_query": map[string]any{
				"search_timeline": map[string]any{
					"timeline": map[string]any{
						"instructions": []any{
							map[string]any{"entries": entries},
						},
					},
				},
			},
		},
	})
	return body
}
