package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestLike_success(t *testing.T) {
	var gotReferer string
	var body map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReferer = r.Header.Get("referer")
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"favorite_tweet":"Done"}}`))
	}))
	defer srv.Close()

	err := c.Like(context.Background(), "tweet1")
	if err != nil {
		t.Fatalf("Like: %v", err)
	}
	if gotReferer != "https://x.com/i/status/tweet1" {
		t.Errorf("referer: got %q", gotReferer)
	}
	vars, _ := body["variables"].(map[string]any)
	if vars["tweet_id"] != "tweet1" {
		t.Errorf("tweet_id: got %v", vars["tweet_id"])
	}
	if body["features"] != nil {
		t.Error("Like must not send features")
	}
}

func TestUnlike_success(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"unfavorite_tweet":"Done"}}`))
	}))
	defer srv.Close()

	err := c.Unlike(context.Background(), "tweet2")
	if err != nil {
		t.Fatalf("Unlike: %v", err)
	}
}

func TestRetweet_success(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"create_retweet":{"retweet_results":{"result":{"rest_id":"rt99"}}}}}`))
	}))
	defer srv.Close()

	id, err := c.Retweet(context.Background(), "tweet3")
	if err != nil {
		t.Fatalf("Retweet: %v", err)
	}
	if id != "rt99" {
		t.Errorf("Retweet: want rt99, got %q", id)
	}
}

func TestUnretweet_bodyShape(t *testing.T) {
	var body map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	err := c.Unretweet(context.Background(), "tweet4")
	if err != nil {
		t.Fatalf("Unretweet: %v", err)
	}
	vars, _ := body["variables"].(map[string]any)
	if vars["tweet_id"] != "tweet4" {
		t.Errorf("tweet_id: got %v", vars["tweet_id"])
	}
	if vars["source_tweet_id"] != "tweet4" {
		t.Errorf("source_tweet_id: got %v", vars["source_tweet_id"])
	}
	// Correction #3: no dark_request, no features
	if vars["dark_request"] != nil {
		t.Error("DeleteRetweet must not have dark_request")
	}
	if body["features"] != nil {
		t.Error("DeleteRetweet must not have features")
	}
}

func TestBookmark_success(t *testing.T) {
	var gotReferer string
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReferer = r.Header.Get("referer")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"tweet_bookmark_put":"Done"}}`))
	}))
	defer srv.Close()

	err := c.Bookmark(context.Background(), "tweet5")
	if err != nil {
		t.Fatalf("Bookmark: %v", err)
	}
	if gotReferer != "https://x.com/i/status/tweet5" {
		t.Errorf("referer: got %q", gotReferer)
	}
}

func TestLike_SendsCorrectBody(t *testing.T) {
	var body map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"favorite_tweet":"Done"}}`))
	}))
	defer srv.Close()

	_ = c.Like(context.Background(), "likeMe")

	vars, ok := body["variables"].(map[string]any)
	if !ok {
		t.Fatal("Like: missing variables")
	}
	if vars["tweet_id"] != "likeMe" {
		t.Errorf("Like: tweet_id want likeMe, got %v", vars["tweet_id"])
	}
}

func TestUnlike_SendsCorrectBody(t *testing.T) {
	var body map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"unfavorite_tweet":"Done"}}`))
	}))
	defer srv.Close()

	_ = c.Unlike(context.Background(), "unlikeMe")

	vars, ok := body["variables"].(map[string]any)
	if !ok {
		t.Fatal("Unlike: missing variables")
	}
	if vars["tweet_id"] != "unlikeMe" {
		t.Errorf("Unlike: tweet_id want unlikeMe, got %v", vars["tweet_id"])
	}
}

func TestRetweet_SendsCorrectBody(t *testing.T) {
	var body map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"create_retweet":{"retweet_results":{"result":{"rest_id":"rt1"}}}}}`))
	}))
	defer srv.Close()

	_, _ = c.Retweet(context.Background(), "rtMe")

	vars, ok := body["variables"].(map[string]any)
	if !ok {
		t.Fatal("Retweet: missing variables")
	}
	if vars["tweet_id"] != "rtMe" {
		t.Errorf("Retweet: tweet_id want rtMe, got %v", vars["tweet_id"])
	}
}

func TestUnretweet_SendsSourceTweetId(t *testing.T) {
	var body map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	_ = c.Unretweet(context.Background(), "srcTweet")

	vars, ok := body["variables"].(map[string]any)
	if !ok {
		t.Fatal("Unretweet: missing variables")
	}
	if vars["tweet_id"] != "srcTweet" {
		t.Errorf("Unretweet: tweet_id want srcTweet, got %v", vars["tweet_id"])
	}
	if vars["source_tweet_id"] != "srcTweet" {
		t.Errorf("Unretweet: source_tweet_id want srcTweet, got %v", vars["source_tweet_id"])
	}
}

func TestBookmark_SendsCorrectVariables(t *testing.T) {
	var body map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"tweet_bookmark_put":"Done"}}`))
	}))
	defer srv.Close()

	_ = c.Bookmark(context.Background(), "bkMe")

	vars, ok := body["variables"].(map[string]any)
	if !ok {
		t.Fatal("Bookmark: missing variables")
	}
	if vars["tweet_id"] != "bkMe" {
		t.Errorf("Bookmark: tweet_id want bkMe, got %v", vars["tweet_id"])
	}
}
