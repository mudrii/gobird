package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/mudrii/gobird/internal/types"
)

// searchTimelineResp builds a mock SearchTimeline response with the given tweet IDs and cursor.
func searchTimelineResp(ids []string, nextCursor string) []byte {
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

func TestSearch_singlePage(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(searchTimelineResp([]string{"1", "2"}, ""))
	}))
	defer srv.Close()

	c.queryIDCache["SearchTimeline"] = "testQID"
	result := c.Search(context.Background(), "hello", nil)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
}

func TestSearch_usesPostWithVariablesInURL(t *testing.T) {
	var gotMethod, gotPath, gotRawQuery string
	var gotBody map[string]any

	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotRawQuery = r.URL.RawQuery
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write(searchTimelineResp([]string{"42"}, ""))
	}))
	defer srv.Close()

	c.queryIDCache["SearchTimeline"] = "testQID"
	c.Search(context.Background(), "golang", nil)

	if gotMethod != "POST" {
		t.Errorf("expected POST, got %q", gotMethod)
	}
	if gotPath != "/i/api/graphql/testQID/SearchTimeline" {
		t.Errorf("unexpected path: %q", gotPath)
	}

	// Variables must be in the URL query string.
	params, _ := url.ParseQuery(gotRawQuery)
	varStr := params.Get("variables")
	if varStr == "" {
		t.Fatal("expected variables in URL query string")
	}
	var vars map[string]any
	json.Unmarshal([]byte(varStr), &vars)
	if vars["rawQuery"] != "golang" {
		t.Errorf("rawQuery: got %v", vars["rawQuery"])
	}
	if vars["product"] != "Latest" {
		t.Errorf("product: got %v", vars["product"])
	}
	if vars["querySource"] != "typed_query" {
		t.Errorf("querySource: got %v", vars["querySource"])
	}

	// POST body must contain features and queryId, NOT variables.
	if gotBody["queryId"] != "testQID" {
		t.Errorf("body queryId: got %v", gotBody["queryId"])
	}
	if gotBody["features"] == nil {
		t.Error("body must contain features")
	}
	if gotBody["variables"] != nil {
		t.Error("body must NOT contain variables (they go in URL)")
	}
}

func TestSearch_defaultProductIsLatest(t *testing.T) {
	var capturedVars map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, _ := url.ParseQuery(r.URL.RawQuery)
		json.Unmarshal([]byte(params.Get("variables")), &capturedVars)
		w.Header().Set("Content-Type", "application/json")
		w.Write(searchTimelineResp(nil, ""))
	}))
	defer srv.Close()

	c.queryIDCache["SearchTimeline"] = "testQID"
	c.Search(context.Background(), "q", nil)
	if capturedVars["product"] != "Latest" {
		t.Errorf("default product should be Latest, got %v", capturedVars["product"])
	}
}

func TestSearch_cursorIncludedWhenPaginating(t *testing.T) {
	var capturedVars map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, _ := url.ParseQuery(r.URL.RawQuery)
		json.Unmarshal([]byte(params.Get("variables")), &capturedVars)
		w.Header().Set("Content-Type", "application/json")
		w.Write(searchTimelineResp(nil, ""))
	}))
	defer srv.Close()

	c.queryIDCache["SearchTimeline"] = "testQID"
	c.Search(context.Background(), "q", &types.SearchOptions{
		FetchOptions: types.FetchOptions{Cursor: "myCursor"},
	})
	if capturedVars["cursor"] != "myCursor" {
		t.Errorf("expected cursor in variables, got %v", capturedVars["cursor"])
	}
}

func TestSearch_404TriggersRefreshAndRetry(t *testing.T) {
	calls := 0
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(searchTimelineResp([]string{"99"}, ""))
	}))
	defer srv.Close()

	c.queryIDCache["SearchTimeline"] = "testQID"
	result := c.Search(context.Background(), "q", nil)
	if !result.Success {
		t.Errorf("expected success after refresh, got: %v", result.Error)
	}
}

func TestIsSearchQueryIDMismatch(t *testing.T) {
	errs := []graphqlError{
		{Message: "something", Extensions: errorExtensions{Code: "GRAPHQL_VALIDATION_FAILED"}},
	}
	if !isSearchQueryIDMismatch(errs) {
		t.Error("expected mismatch detected")
	}

	errs2 := []graphqlError{
		{Message: "rawQuery must be defined", Path: []any{"rawQuery"}},
	}
	if !isSearchQueryIDMismatch(errs2) {
		t.Error("expected mismatch detected for rawQuery must be defined")
	}

	errs3 := []graphqlError{
		{Message: "unrelated error"},
	}
	if isSearchQueryIDMismatch(errs3) {
		t.Error("should not detect mismatch for unrelated error")
	}
}

func TestGetAllSearchResults_pagination(t *testing.T) {
	calls := 0
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		switch calls {
		case 1:
			w.Write(searchTimelineResp([]string{"1", "2"}, "cursor2"))
		case 2:
			w.Write(searchTimelineResp([]string{"3"}, ""))
		}
	}))
	defer srv.Close()

	c.queryIDCache["SearchTimeline"] = "testQID"
	result := c.GetAllSearchResults(context.Background(), "go", &types.SearchOptions{
		FetchOptions: types.FetchOptions{PageDelayMs: 0},
	})
	if !result.Success {
		t.Fatalf("expected success, got: %v", result.Error)
	}
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items across pages, got %d", len(result.Items))
	}
}
