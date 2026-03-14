package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(handler http.Handler) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	c := New("tok", "ct0", &Options{
		HTTPClient: &http.Client{},
		QueryIDCache: map[string]string{
			"CreateTweet": "testQueryID",
		},
		RequestsPerSecond: -1,
	})
	// Redirect all requests to the test server by using a custom transport.
	c.httpClient = &http.Client{
		Transport: redirectTransport(srv.URL),
	}
	// Prevent real HTTP scraping in tests.
	c.scraper = func(_ context.Context) map[string]string { return nil }
	return c, srv
}

type redirectTransport string

func (base redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the host portion to the test server, keep path+query.
	newURL := string(base) + req.URL.Path
	if req.URL.RawQuery != "" {
		newURL += "?" + req.URL.RawQuery
	}
	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
	if err != nil {
		return nil, err
	}
	newReq.Header = req.Header
	return http.DefaultTransport.RoundTrip(newReq)
}

func tweetCreateResp(id string) []byte {
	b, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"create_tweet": map[string]any{
				"tweet_results": map[string]any{
					"result": map[string]any{
						"rest_id": id,
					},
				},
			},
		},
	})
	return b
}

func TestTweet_success(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(tweetCreateResp("1234567890"))
	}))
	defer srv.Close()

	id, err := c.Tweet(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("Tweet: %v", err)
	}
	if id != "1234567890" {
		t.Errorf("Tweet: want id 1234567890, got %q", id)
	}
}

func TestTweet_refererHeader(t *testing.T) {
	var gotReferer string
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReferer = r.Header.Get("referer")
		w.Header().Set("Content-Type", "application/json")
		w.Write(tweetCreateResp("42"))
	}))
	defer srv.Close()

	_, _ = c.Tweet(context.Background(), "hello")
	if gotReferer != "https://x.com/compose/post" {
		t.Errorf("referer: want https://x.com/compose/post, got %q", gotReferer)
	}
}

func TestTweet_bodyShape(t *testing.T) {
	var body map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.Write(tweetCreateResp("1"))
	}))
	defer srv.Close()

	_, _ = c.Tweet(context.Background(), "test text")

	vars, ok := body["variables"].(map[string]any)
	if !ok {
		t.Fatal("missing variables")
	}
	if vars["tweet_text"] != "test text" {
		t.Errorf("tweet_text: got %v", vars["tweet_text"])
	}
	if vars["dark_request"] != false {
		t.Errorf("dark_request: got %v", vars["dark_request"])
	}
	if body["features"] == nil {
		t.Error("missing features")
	}
	if body["queryId"] == nil {
		t.Error("missing queryId")
	}
}

func TestReply_addsReplyField(t *testing.T) {
	var body map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.Write(tweetCreateResp("99"))
	}))
	defer srv.Close()

	_, _ = c.Reply(context.Background(), "reply text", "111")
	vars, _ := body["variables"].(map[string]any)
	reply, ok := vars["reply"].(map[string]any)
	if !ok {
		t.Fatal("missing reply field in variables")
	}
	if reply["in_reply_to_tweet_id"] != "111" {
		t.Errorf("in_reply_to_tweet_id: got %v", reply["in_reply_to_tweet_id"])
	}
}

func TestTweet_fallbackOn404(t *testing.T) {
	calls := 0
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			http.Error(w, `{"errors":[{"message":"not found"}]}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(tweetCreateResp("final"))
	}))
	defer srv.Close()

	id, err := c.Tweet(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Tweet: %v", err)
	}
	if id != "final" {
		t.Errorf("want 'final', got %q", id)
	}
	if calls < 3 {
		t.Errorf("expected at least 3 calls, got %d", calls)
	}
}

func TestTweet_code226FallbackToStatusUpdate(t *testing.T) {
	calls := 0
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if strings.Contains(r.URL.Path, "statuses/update") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id_str":"statusID"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// Return error code 226
		w.Write([]byte(`{"errors":[{"message":"automated","extensions":{"code":"226"}}]}`))
	}))
	defer srv.Close()

	id, err := c.Tweet(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Tweet: %v", err)
	}
	if id != "statusID" {
		t.Errorf("want statusID, got %q", id)
	}
}

func TestExtractCreateTweetID_missingField(t *testing.T) {
	_, err := extractCreateTweetID([]byte(`{}`))
	if err == nil {
		t.Error("expected error for empty rest_id")
	}
}

func TestTryStatusUpdateFallback_idStr(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id_str":"abc123"}`)
	}))
	defer srv.Close()

	id, err := c.tryStatusUpdateFallback(context.Background(), "text", "")
	if err != nil {
		t.Fatalf("tryStatusUpdateFallback: %v", err)
	}
	if id != "abc123" {
		t.Errorf("want abc123, got %q", id)
	}
}

func TestTryStatusUpdateFallback_numericID(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":99887766}`)
	}))
	defer srv.Close()

	id, err := c.tryStatusUpdateFallback(context.Background(), "text", "")
	if err != nil {
		t.Fatalf("tryStatusUpdateFallback: %v", err)
	}
	if id != "99887766" {
		t.Errorf("want 99887766, got %q", id)
	}
}

func TestCreateTweet_Success(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(tweetCreateResp("tweet42"))
	}))
	defer srv.Close()

	id, err := c.Tweet(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("CreateTweet_Success: %v", err)
	}
	if id != "tweet42" {
		t.Errorf("CreateTweet_Success: want tweet42, got %q", id)
	}
}

func TestCreateTweet_WithMedia(t *testing.T) {
	var body map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.Write(tweetCreateResp("mediatweet"))
	}))
	defer srv.Close()

	_, _ = c.TweetWithMedia(context.Background(), "media post", []string{"111", "222"})

	vars, ok := body["variables"].(map[string]any)
	if !ok {
		t.Fatal("missing variables")
	}
	media, ok := vars["media"].(map[string]any)
	if !ok {
		t.Fatal("missing media in variables")
	}
	entities, ok := media["media_entities"].([]any)
	if !ok {
		t.Fatal("missing media_entities")
	}
	if len(entities) != 2 {
		t.Errorf("want 2 media entities, got %d", len(entities))
	}
	first, ok := entities[0].(map[string]any)
	if !ok {
		t.Fatal("media entity is not a map")
	}
	if first["media_id"] != "111" {
		t.Errorf("first media_id: want 111, got %v", first["media_id"])
	}
}

func TestCreateTweet_FallbackOnBadRequest(t *testing.T) {
	// 400 is NOT a 404, so the fallback chain does NOT trigger.
	// The first call returns 400 and the function returns an error.
	calls := 0
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, `{"errors":[{"message":"bad request"}]}`, http.StatusBadRequest)
	}))
	defer srv.Close()

	_, err := c.Tweet(context.Background(), "bad")
	if err == nil {
		t.Fatal("expected error on 400")
	}
	// 400 is not retried — only one call expected.
	if calls != 1 {
		t.Errorf("expected 1 call on 400 (no retry), got %d", calls)
	}
}

func TestReply_SetsInReplyToId(t *testing.T) {
	var body map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.Write(tweetCreateResp("replyID"))
	}))
	defer srv.Close()

	_, _ = c.Reply(context.Background(), "replying", "parentTweet99")

	vars, ok := body["variables"].(map[string]any)
	if !ok {
		t.Fatal("missing variables")
	}
	reply, ok := vars["reply"].(map[string]any)
	if !ok {
		t.Fatal("missing reply field in variables")
	}
	if reply["in_reply_to_tweet_id"] != "parentTweet99" {
		t.Errorf("in_reply_to_tweet_id: want parentTweet99, got %v", reply["in_reply_to_tweet_id"])
	}
}
