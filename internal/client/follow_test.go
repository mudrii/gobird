package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"io"
	"strings"
	"testing"
)

func TestFollow_restSuccess(t *testing.T) {
	var gotURL string
	var gotBody string
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id_str":"user99"}`))
	}))
	defer srv.Close()

	err := c.Follow(context.Background(), "user99")
	if err != nil {
		t.Fatalf("Follow: %v", err)
	}
	if !strings.Contains(gotURL, "friendships/create") {
		t.Errorf("expected friendships/create endpoint, got %q", gotURL)
	}
	vals, _ := url.ParseQuery(gotBody)
	if vals.Get("user_id") != "user99" {
		t.Errorf("user_id: got %q", vals.Get("user_id"))
	}
	if vals.Get("skip_status") != "true" {
		t.Errorf("skip_status: got %q", vals.Get("skip_status"))
	}
}

func TestFollow_alreadyFollowing_isSuccess(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return code 160 = already following
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"errors":[{"code":160,"message":"already following"}]}`))
	}))
	defer srv.Close()

	err := c.followViaREST(context.Background(), srv.URL+"/1.1/friendships/create.json", "u1")
	if err != nil {
		t.Errorf("already-following should be success, got: %v", err)
	}
}

func TestFollow_blocked_returnsError(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"errors":[{"code":162,"message":"blocked"}]}`))
	}))
	defer srv.Close()

	err := c.followViaREST(context.Background(), srv.URL+"/1.1/friendships/create.json", "u2")
	if !isFollowBlocked(err) {
		t.Errorf("expected blocked error, got: %v", err)
	}
}

func TestFollow_notFound_returnsError(t *testing.T) {
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":[{"code":108,"message":"not found"}]}`))
	}))
	defer srv.Close()

	err := c.followViaREST(context.Background(), srv.URL+"/1.1/friendships/create.json", "u3")
	if !isFollowNotFound(err) {
		t.Errorf("expected not-found error, got: %v", err)
	}
}

func TestFollowViaGraphQL_bodyShape(t *testing.T) {
	var gotBody map[string]any
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	_ = c.followViaGraphQL(context.Background(), "gqlUser1")
	vars, _ := gotBody["variables"].(map[string]any)
	if vars["user_id"] != "gqlUser1" {
		t.Errorf("user_id: got %v", vars["user_id"])
	}
	if gotBody["features"] != nil {
		t.Error("CreateFriendship must not send features")
	}
}

func TestUnfollow_success(t *testing.T) {
	var gotURL string
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id_str":"u4"}`))
	}))
	defer srv.Close()

	err := c.Unfollow(context.Background(), "u4")
	if err != nil {
		t.Fatalf("Unfollow: %v", err)
	}
	if !strings.Contains(gotURL, "friendships/destroy") {
		t.Errorf("expected friendships/destroy endpoint, got %q", gotURL)
	}
}

func TestFollow_RESTFirst(t *testing.T) {
	// Verify the very first request hits the REST friendships/create endpoint.
	var firstURL string
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if firstURL == "" {
			firstURL = r.URL.Path
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id_str":"u99"}`))
	}))
	defer srv.Close()

	_ = c.Follow(context.Background(), "u99")

	if !strings.Contains(firstURL, "friendships/create") {
		t.Errorf("Follow_RESTFirst: first request should be friendships/create, got %q", firstURL)
	}
}

func TestFollow_ErrorCode160_Success(t *testing.T) {
	// Error code 160 in a 2xx body means already-following → treat as success.
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// 200 OK body with errors array containing code 160.
		w.Write([]byte(`{"errors":[{"code":160,"message":"already following"}]}`))
	}))
	defer srv.Close()

	err := c.followViaREST(context.Background(), srv.URL+"/1.1/friendships/create.json", "u160")
	if err != nil {
		t.Errorf("code 160 in body should be success, got: %v", err)
	}
}

func TestFollow_ErrorCode162_Blocked(t *testing.T) {
	// Error code 162 = blocked → should surface as a followError.
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"errors":[{"code":162,"message":"blocked"}]}`))
	}))
	defer srv.Close()

	err := c.followViaREST(context.Background(), srv.URL+"/1.1/friendships/create.json", "u162")
	if !isFollowBlocked(err) {
		t.Errorf("code 162 should be follow-blocked error, got: %v", err)
	}
}

func TestUnfollow_SendsUserId(t *testing.T) {
	var gotBody string
	c, srv := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id_str":"u5"}`))
	}))
	defer srv.Close()

	_ = c.Unfollow(context.Background(), "u5")

	vals, _ := url.ParseQuery(gotBody)
	if vals.Get("user_id") != "u5" {
		t.Errorf("Unfollow user_id: want u5, got %q", vals.Get("user_id"))
	}
}
