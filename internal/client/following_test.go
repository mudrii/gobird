package client

import (
	"context"
	"net/http"
	"testing"

	"github.com/mudrii/gobird/internal/testutil"
)

const followingGraphQLJSON = `{
	"data": {
		"user": {
			"result": {
				"timeline": {
					"timeline": {
						"instructions": [
							{
								"type": "TimelineAddEntries",
								"entries": [
									{
										"entryId": "user-200",
										"sortIndex": "2",
										"content": {
											"entryType": "TimelineTimelineItem",
											"itemContent": {
												"__typename": "TimelineUser",
												"user_results": {
													"result": {
														"__typename": "User",
														"rest_id": "200",
														"is_blue_verified": false,
														"legacy": {
															"screen_name": "follower1",
															"name": "Follower One",
															"description": "",
															"followers_count": 5,
															"friends_count": 3,
															"profile_image_url_https": "",
															"created_at": ""
														}
													}
												}
											}
										}
									},
									{
										"entryId": "cursor-bottom",
										"sortIndex": "0",
										"content": {
											"entryType": "TimelineTimelineCursor",
											"cursorType": "Bottom",
											"value": ""
										}
									}
								]
							}
						]
					}
				}
			}
		}
	}
}`

func TestGetFollowing_Basic(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, followingGraphQLJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{
		"Following": "mWYeougg_ocJS2Vr1Vt28w",
	})
	result, err := c.GetFollowing(context.Background(), "user-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Items) != 1 {
		t.Fatalf("want 1 user, got %d", len(result.Items))
	}
	if result.Items[0].ID != "200" {
		t.Errorf("want user ID=200, got %q", result.Items[0].ID)
	}
	if result.Items[0].Username != "follower1" {
		t.Errorf("want username=follower1, got %q", result.Items[0].Username)
	}
}

func TestGetFollowers_Basic(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, followingGraphQLJSON))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{
		"Followers": "SFYY3WsgwjlXSLlfnEUE4A",
	})
	result, err := c.GetFollowers(context.Background(), "user-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success")
	}
	if len(result.Items) != 1 {
		t.Fatalf("want 1 user, got %d", len(result.Items))
	}
}

func TestGetFollowing_404TriggersRefresh(t *testing.T) {
	callCount := 0
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"errors":[{"message":"Not Found"}]}`))
	}))
	defer srv.Close()

	c := newTestClientWith(srv.URL, map[string]string{
		"Following": "mWYeougg_ocJS2Vr1Vt28w",
	})
	_, _ = c.GetFollowing(context.Background(), "user-x", nil)
	// Verify that multiple attempts were made (initial + refresh + retry).
	if callCount < 2 {
		t.Errorf("want at least 2 calls for 404+refresh pattern, got %d", callCount)
	}
}

func TestParseFollowResponse(t *testing.T) {
	page, err := parseFollowResponse([]byte(followingGraphQLJSON), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !page.Success {
		t.Fatal("expected success")
	}
	if len(page.Items) != 1 {
		t.Fatalf("want 1 user, got %d", len(page.Items))
	}
	if page.Items[0].Username != "follower1" {
		t.Errorf("want username=follower1, got %q", page.Items[0].Username)
	}
}

func TestParseRESTFollowResponse(t *testing.T) {
	const restJSON = `{
		"users": [
			{
				"id_str": "301",
				"screen_name": "restuser",
				"name": "REST User",
				"followers_count": 10,
				"friends_count": 5
			}
		],
		"next_cursor_str": "0"
	}`
	page, err := parseRESTFollowResponse([]byte(restJSON), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("want 1 user, got %d", len(page.Items))
	}
	if page.Items[0].ID != "301" {
		t.Errorf("want ID=301, got %q", page.Items[0].ID)
	}
	// next_cursor_str "0" should be normalized to "".
	if page.NextCursor != "" {
		t.Errorf("want empty NextCursor for cursor '0', got %q", page.NextCursor)
	}
}

func TestParseRESTFollowResponse_WithNextCursor(t *testing.T) {
	const restJSON = `{
		"users": [{"id_str": "400", "screen_name": "u400"}],
		"next_cursor_str": "12345"
	}`
	page, err := parseRESTFollowResponse([]byte(restJSON), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.NextCursor != "12345" {
		t.Errorf("want NextCursor=12345, got %q", page.NextCursor)
	}
}
