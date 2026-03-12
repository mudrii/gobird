package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

// homeTimelineResp builds a mock HomeTimeline response.
func homeTimelineResp(ids []string, nextCursor string) []byte {
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
			"home": map[string]any{
				"home_timeline_urt": map[string]any{
					"instructions": []any{
						map[string]any{"entries": entries},
					},
				},
			},
		},
	})
	return body
}

func TestGetHomeTimeline_basic(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(homeTimelineResp([]string{"10", "11"}, ""))
	}))
	defer srv.Close()

	c.queryIDCache["HomeTimeline"] = "homeQID"
	result := c.GetHomeTimeline(context.Background(), nil)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
}

func TestGetHomeTimeline_noNextCursorReturned(t *testing.T) {
	// Correction #50: HomeTimeline never returns nextCursor to caller.
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(homeTimelineResp([]string{"1"}, "someCursor"))
	}))
	defer srv.Close()

	c.queryIDCache["HomeTimeline"] = "homeQID"
	result := c.GetHomeTimeline(context.Background(), nil)
	if result.NextCursor != "" {
		t.Errorf("HomeTimeline should not return nextCursor, got %q", result.NextCursor)
	}
}

func TestGetHomeTimeline_usesGetMethod(t *testing.T) {
	var gotMethod string
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		w.Write(homeTimelineResp(nil, ""))
	}))
	defer srv.Close()

	c.queryIDCache["HomeTimeline"] = "homeQID"
	c.GetHomeTimeline(context.Background(), nil)
	if gotMethod != "GET" {
		t.Errorf("expected GET method, got %q", gotMethod)
	}
}

func TestGetHomeTimeline_variables(t *testing.T) {
	var capturedVars map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, _ := url.ParseQuery(r.URL.RawQuery)
		json.Unmarshal([]byte(params.Get("variables")), &capturedVars)
		w.Header().Set("Content-Type", "application/json")
		w.Write(homeTimelineResp(nil, ""))
	}))
	defer srv.Close()

	c.queryIDCache["HomeTimeline"] = "homeQID"
	c.GetHomeTimeline(context.Background(), nil)

	// Correction #61: must include latestControlAvailable and requestContext.
	if capturedVars["latestControlAvailable"] != true {
		t.Errorf("latestControlAvailable should be true, got %v", capturedVars["latestControlAvailable"])
	}
	if capturedVars["requestContext"] != "launch" {
		t.Errorf("requestContext should be 'launch', got %v", capturedVars["requestContext"])
	}
	if capturedVars["withCommunity"] != true {
		t.Errorf("withCommunity should be true, got %v", capturedVars["withCommunity"])
	}
	if capturedVars["includePromotedContent"] != true {
		t.Errorf("includePromotedContent should be true, got %v", capturedVars["includePromotedContent"])
	}
}

func TestGetHomeTimeline_queryUnspecifiedTriggersRefresh(t *testing.T) {
	// Correction #77: case-insensitive "query: unspecified" triggers refresh.
	calls := 0
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			// Return GraphQL error matching /query:\s*unspecified/i.
			w.Write([]byte(`{"errors":[{"message":"Query: Unspecified"}]}`))
			return
		}
		w.Write(homeTimelineResp([]string{"5"}, ""))
	}))
	defer srv.Close()

	c.queryIDCache["HomeTimeline"] = "homeQID"
	result := c.GetHomeTimeline(context.Background(), nil)
	if !result.Success {
		t.Errorf("expected success after refresh, got: %v", result.Error)
	}
}

func TestGetHomeLatestTimeline_basic(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(homeTimelineResp([]string{"20"}, ""))
	}))
	defer srv.Close()

	c.queryIDCache["HomeLatestTimeline"] = "homeLatestQID"
	result := c.GetHomeLatestTimeline(context.Background(), nil)
	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
}

func TestGetHomeLatestTimeline_noNextCursor(t *testing.T) {
	// Correction #50: also applies to HomeLatestTimeline.
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(homeTimelineResp([]string{"1"}, "aCursor"))
	}))
	defer srv.Close()

	c.queryIDCache["HomeLatestTimeline"] = "homeLatestQID"
	result := c.GetHomeLatestTimeline(context.Background(), nil)
	if result.NextCursor != "" {
		t.Errorf("HomeLatestTimeline should not return nextCursor, got %q", result.NextCursor)
	}
}

func TestQueryUnspecifiedRe(t *testing.T) {
	// Correction #77: case-insensitive, allows optional whitespace.
	cases := []struct {
		msg   string
		match bool
	}{
		{"Query: Unspecified", true},
		{"query: unspecified", true},
		{"QUERY:  UNSPECIFIED", true},
		{"query:unspecified", true},
		{"unrelated error", false},
		{"Query timeout", false},
	}
	for _, c := range cases {
		got := queryUnspecifiedRe.MatchString(c.msg)
		if got != c.match {
			t.Errorf("queryUnspecifiedRe.MatchString(%q) = %v, want %v", c.msg, got, c.match)
		}
	}
}
