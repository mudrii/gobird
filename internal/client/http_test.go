package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mudrii/gobird/internal/testutil"
)

func newBareTestClient(baseURL string) *Client {
	transport := testutil.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
		testReq, _ := http.NewRequestWithContext(r.Context(), r.Method, baseURL+r.URL.RequestURI(), r.Body)
		for k, vs := range r.Header {
			testReq.Header[k] = vs
		}
		return http.DefaultTransport.RoundTrip(testReq)
	})
	c := New("fake-auth", "fake-ct0", &Options{
		HTTPClient: &http.Client{Transport: transport},
	})
	c.scraper = func(_ context.Context) map[string]string { return nil }
	return c
}

func TestDoGET_Success(t *testing.T) {
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("doGET: want GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newBareTestClient(srv.URL)
	body, err := c.doGET(context.Background(), srv.URL+"/test", c.getJsonHeaders())
	if err != nil {
		t.Fatalf("doGET: unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("doGET: invalid JSON response: %v", err)
	}
	if result["ok"] != true {
		t.Errorf("doGET: want ok=true, got %v", result["ok"])
	}
}

func TestDoGET_404(t *testing.T) {
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"errors":[{"message":"not found"}]}`))
	}))
	defer srv.Close()

	c := newBareTestClient(srv.URL)
	_, err := c.doGET(context.Background(), srv.URL+"/missing", c.getJsonHeaders())
	if err == nil {
		t.Fatal("doGET: expected error for 404, got nil")
	}
	if !is404(err) {
		t.Errorf("doGET: expected is404=true for 404, got false (err=%v)", err)
	}
}

func TestDoPOSTJSON_Success(t *testing.T) {
	var received map[string]any
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("doPOSTJSON: want POST, got %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&received)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"result":"ok"}`))
	}))
	defer srv.Close()

	c := newBareTestClient(srv.URL)
	payload := map[string]any{"key": "value"}
	body, err := c.doPOSTJSON(context.Background(), srv.URL+"/post", c.getJsonHeaders(), payload)
	if err != nil {
		t.Fatalf("doPOSTJSON: unexpected error: %v", err)
	}
	if received["key"] != "value" {
		t.Errorf("doPOSTJSON: sent key=value, received %v", received["key"])
	}
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("doPOSTJSON: invalid JSON response: %v", err)
	}
	if result["result"] != "ok" {
		t.Errorf("doPOSTJSON: want result=ok, got %v", result["result"])
	}
}

func TestFetchWithRetry_SuccessFirstAttempt(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newBareTestClient(srv.URL)
	body, err := c.fetchWithRetry(context.Background(), srv.URL+"/data", c.getJsonHeaders())
	if err != nil {
		t.Fatalf("fetchWithRetry: unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("fetchWithRetry: want 1 call on success, got %d", calls)
	}
	if len(body) == 0 {
		t.Error("fetchWithRetry: expected non-empty body")
	}
}

func TestFetchWithRetry_SuccessOnRetry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls < 3 {
			// 500 is retryable.
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"error":"server error"}`))
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newBareTestClient(srv.URL)
	// Use a context that we cancel after the test to avoid long delays.
	body, err := c.fetchWithRetry(context.Background(), srv.URL+"/data", c.getJsonHeaders())
	if err != nil {
		t.Fatalf("fetchWithRetry: unexpected error on retry: %v", err)
	}
	if calls != 3 {
		t.Errorf("fetchWithRetry: want 3 calls (2 failures + success), got %d", calls)
	}
	if len(body) == 0 {
		t.Error("fetchWithRetry: expected non-empty body on success")
	}
}

func TestFetchWithRetry_AllFail(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		// 429 is retryable — all 3 attempts fail.
		w.Header().Set("retry-after", "0")
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer srv.Close()

	c := newBareTestClient(srv.URL)
	_, err := c.fetchWithRetry(context.Background(), srv.URL+"/data", c.getJsonHeaders())
	if err == nil {
		t.Fatal("fetchWithRetry: expected error when all attempts fail")
	}
	// maxRetries=2 means 3 total attempts.
	if calls != 3 {
		t.Errorf("fetchWithRetry: want 3 total attempts, got %d", calls)
	}
}

func TestParseGraphQLErrors_WithErrors(t *testing.T) {
	body := []byte(`{"errors":[{"message":"Something went wrong","extensions":{"code":"ERR"}}]}`)
	errs := parseGraphQLErrors(body)
	if len(errs) == 0 {
		t.Fatal("parseGraphQLErrors: expected errors, got none")
	}
	if errs[0].Message != "Something went wrong" {
		t.Errorf("parseGraphQLErrors: want message 'Something went wrong', got %q", errs[0].Message)
	}
	if errs[0].Extensions.Code != "ERR" {
		t.Errorf("parseGraphQLErrors: want code 'ERR', got %q", errs[0].Extensions.Code)
	}
}

func TestParseGraphQLErrors_NoErrors(t *testing.T) {
	body := []byte(`{"data":{"user":{"id":"123"}}}`)
	errs := parseGraphQLErrors(body)
	if len(errs) != 0 {
		t.Errorf("parseGraphQLErrors: expected no errors, got %d", len(errs))
	}
}
