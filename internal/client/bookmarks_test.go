package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestUnbookmark_success(t *testing.T) {
	var gotReferer string
	var body map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReferer = r.Header.Get("referer")
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"tweet_bookmark_delete":"Done"}}`))
	}))
	defer srv.Close()

	err := c.Unbookmark(context.Background(), "bktweet1")
	if err != nil {
		t.Fatalf("Unbookmark: %v", err)
	}
	if gotReferer != "https://x.com/i/status/bktweet1" {
		t.Errorf("referer: got %q", gotReferer)
	}
	vars, _ := body["variables"].(map[string]any)
	if vars["tweet_id"] != "bktweet1" {
		t.Errorf("tweet_id: got %v", vars["tweet_id"])
	}
	if body["features"] != nil {
		t.Error("DeleteBookmark must not send features")
	}
}
