package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mudrii/gobird/internal/testutil"
)

type closeErrorBody struct {
	*strings.Reader
	closeErr error
}

func (b *closeErrorBody) Close() error {
	return b.closeErr
}

func newBareTestClient(baseURL string) *Client {
	transport := testutil.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
		testReq, _ := http.NewRequestWithContext(r.Context(), r.Method, baseURL+r.URL.RequestURI(), r.Body)
		for k, vs := range r.Header {
			testReq.Header[k] = vs
		}
		return http.DefaultTransport.RoundTrip(testReq)
	})
	c := New("fake-auth", "fake-ct0", &Options{
		HTTPClient:        &http.Client{Transport: transport},
		RequestsPerSecond: -1,
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
	body, err := c.doGET(context.Background(), srv.URL+"/test", c.getJSONHeaders())
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
	_, err := c.doGET(context.Background(), srv.URL+"/missing", c.getJSONHeaders())
	if err == nil {
		t.Fatal("doGET: expected error for 404, got nil")
	}
	if !is404(err) {
		t.Errorf("doGET: expected is404=true for 404, got false (err=%v)", err)
	}
}

func TestDo_CloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	c := New("tok", "ct0", &Options{
		HTTPClient: &http.Client{
			Transport: testutil.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Header:     make(http.Header),
					Body: &closeErrorBody{
						Reader:   strings.NewReader(`{"ok":true}`),
						closeErr: closeErr,
					},
				}, nil
			}),
		},
		RequestsPerSecond: -1,
	})

	_, err := c.doGET(context.Background(), "https://example.com/test", c.getJSONHeaders())
	if err == nil {
		t.Fatal("expected error due to close failure")
	}
	if !strings.Contains(err.Error(), "close body") {
		t.Fatalf("unexpected error: %v", err)
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
	body, err := c.doPOSTJSON(context.Background(), srv.URL+"/post", c.getJSONHeaders(), payload)
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
	body, err := c.fetchWithRetry(context.Background(), srv.URL+"/data", c.getJSONHeaders())
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
	body, err := c.fetchWithRetry(context.Background(), srv.URL+"/data", c.getJSONHeaders())
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
	_, err := c.fetchWithRetry(context.Background(), srv.URL+"/data", c.getJSONHeaders())
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

func TestDoGET_ContextCancelled(t *testing.T) {
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This handler should not be reached because context is already cancelled.
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newBareTestClient(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before the request

	_, err := c.doGET(ctx, srv.URL+"/test", c.getJSONHeaders())
	if err == nil {
		t.Fatal("doGET: expected error for cancelled context, got nil")
	}
}

func TestDoPOSTJSON_InvalidBody(t *testing.T) {
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := newBareTestClient(srv.URL)
	// A channel cannot be marshaled to JSON.
	body := map[string]any{"bad": make(chan int)}
	_, err := c.doPOSTJSON(context.Background(), srv.URL+"/post", c.getJSONHeaders(), body)
	if err == nil {
		t.Fatal("doPOSTJSON: expected marshal error for un-marshalable body, got nil")
	}
}

func TestGraphqlURL_Encoding(t *testing.T) {
	// graphqlURL just concatenates; verify no double-encoding happens.
	u := graphqlURL("CreateTweet", "testQID")
	wantSuffix := "/testQID/CreateTweet"
	if len(u) < len(wantSuffix) || u[len(u)-len(wantSuffix):] != wantSuffix {
		t.Errorf("graphqlURL: want suffix %q, got %q", wantSuffix, u)
	}
}

func TestDoGET_ServerError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"500", 500},
		{"502", 502},
		{"503", 503},
		{"400", 400},
		{"401", 401},
		{"403", 403},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := testutil.NewTestServer(testutil.StaticHandler(tt.statusCode, `{"error":"test"}`))
			defer srv.Close()

			c := newBareTestClient(srv.URL)
			_, err := c.doGET(context.Background(), srv.URL+"/err", c.getJSONHeaders())
			if err == nil {
				t.Fatalf("expected error for HTTP %d", tt.statusCode)
			}
			var he *httpError
			if !errors.As(err, &he) {
				t.Fatalf("expected *httpError, got %T", err)
			}
			if he.StatusCode != tt.statusCode {
				t.Errorf("status: want %d, got %d", tt.statusCode, he.StatusCode)
			}
		})
	}
}

func TestDoGET_EmptyResponseBody(t *testing.T) {
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := newBareTestClient(srv.URL)
	body, err := c.doGET(context.Background(), srv.URL+"/empty", c.getJSONHeaders())
	if err != nil {
		t.Fatalf("doGET: unexpected error: %v", err)
	}
	if len(body) != 0 {
		t.Errorf("expected empty body, got %d bytes", len(body))
	}
}

func TestDoPOSTForm_Success(t *testing.T) {
	var gotContentType string
	var gotBody string
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newBareTestClient(srv.URL)
	body, err := c.doPOSTForm(context.Background(), srv.URL+"/form", c.getBaseHeaders(), "key=val&other=123")
	if err != nil {
		t.Fatalf("doPOSTForm: %v", err)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("content-type: want application/x-www-form-urlencoded, got %q", gotContentType)
	}
	if gotBody != "key=val&other=123" {
		t.Errorf("body: want key=val&other=123, got %q", gotBody)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response")
	}
}

func TestDoPOSTForm_ServerError(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(500, `{"error":"boom"}`))
	defer srv.Close()

	c := newBareTestClient(srv.URL)
	_, err := c.doPOSTForm(context.Background(), srv.URL+"/form", c.getBaseHeaders(), "data=1")
	if err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestIs404_edgeCases(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"non-httpError", errors.New("other"), false},
		{"httpError 200", &httpError{StatusCode: 200, Body: ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := is404(tt.err)
			if got != tt.want {
				t.Errorf("is404(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestHTTPStatusCode(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", &httpError{StatusCode: 429, Body: "rate limited"})
	status, ok := HTTPStatusCode(err)
	if !ok {
		t.Fatal("HTTPStatusCode should recognize wrapped httpError")
	}
	if status != 429 {
		t.Fatalf("HTTPStatusCode() = %d, want 429", status)
	}
}

func TestHTTPStatusCode_NoHTTPError(t *testing.T) {
	status, ok := HTTPStatusCode(errors.New("plain error"))
	if ok {
		t.Fatalf("HTTPStatusCode should reject non-httpError, got status %d", status)
	}
}

func TestHttpError_ErrorMessage(t *testing.T) {
	he := &httpError{StatusCode: 429, Body: "rate limited"}
	msg := he.Error()
	if !strings.Contains(msg, "429") {
		t.Errorf("error should contain status code, got: %s", msg)
	}
	if !strings.Contains(msg, "rate limited") {
		t.Errorf("error should contain body, got: %s", msg)
	}
}

func TestHttpError_LongBodyTruncated(t *testing.T) {
	longBody := strings.Repeat("x", 300)
	he := &httpError{StatusCode: 500, Body: longBody}
	msg := he.Error()
	if len(msg) > 250 {
		// 200 chars body + "HTTP 500: " prefix + truncation marker
		if !strings.Contains(msg, "…") {
			t.Error("long body should be truncated with ellipsis")
		}
	}
}

func TestRetryableStatus(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
	}
	for _, tt := range tests {
		got := retryableStatus(tt.code)
		if got != tt.want {
			t.Errorf("retryableStatus(%d) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestFetchWithRetry_NonRetryableErrorNoRetry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(403)
		w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer srv.Close()

	c := newBareTestClient(srv.URL)
	_, err := c.fetchWithRetry(context.Background(), srv.URL+"/data", c.getJSONHeaders())
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if calls != 1 {
		t.Errorf("403 should not be retried, want 1 call, got %d", calls)
	}
}

func TestFetchWithRetry_ContextCancelledDuringRetry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("retry-after", "0")
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"fail"}`))
	}))
	defer srv.Close()

	c := newBareTestClient(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after first call completes — the retry delay select should pick up cancellation.
	go func() {
		for calls.Load() < 1 {
			runtime.Gosched()
		}
		cancel()
	}()

	_, err := c.fetchWithRetry(ctx, srv.URL+"/data", c.getJSONHeaders())
	if err == nil {
		t.Fatal("expected error when context cancelled during retry")
	}
}

func TestFetchWithRetry_RetryAfterHeader(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("retry-after", "0")
			w.WriteHeader(429)
			w.Write([]byte(`{"error":"rate limited"}`))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newBareTestClient(srv.URL)
	body, err := c.fetchWithRetry(context.Background(), srv.URL+"/data", c.getJSONHeaders())
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if calls != 2 {
		t.Errorf("want 2 calls, got %d", calls)
	}
	if len(body) == 0 {
		t.Error("expected non-empty body")
	}
}

func TestFetchWithRetry_TransportErrorRetries(t *testing.T) {
	var calls atomic.Int32
	c := New("tok", "ct0", &Options{
		HTTPClient: &http.Client{
			Transport: testutil.RoundTripFunc(func(r *http.Request) (*http.Response, error) {
				if calls.Add(1) < 3 {
					return nil, errors.New("connection reset by peer")
				}
				return &http.Response{
					StatusCode: 200,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				}, nil
			}),
		},
		RequestsPerSecond: -1,
	})

	body, err := c.fetchWithRetry(context.Background(), "https://example.com/data", c.getJSONHeaders())
	if err != nil {
		t.Fatalf("expected retry success after transport errors, got: %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("want 3 attempts, got %d", calls.Load())
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty body")
	}
}

func TestParseGraphQLErrors_InvalidJSON(t *testing.T) {
	errs := parseGraphQLErrors([]byte(`not json`))
	if len(errs) != 0 {
		t.Errorf("invalid JSON should return nil errors, got %d", len(errs))
	}
}

func TestParseGraphQLErrors_EmptyErrorsArray(t *testing.T) {
	errs := parseGraphQLErrors([]byte(`{"errors":[]}`))
	if len(errs) != 0 {
		t.Errorf("empty errors array should return empty slice, got %d", len(errs))
	}
}

func TestParseGraphQLErrors_NilBody(t *testing.T) {
	errs := parseGraphQLErrors(nil)
	if len(errs) != 0 {
		t.Errorf("nil body should return nil, got %d", len(errs))
	}
}

func TestGraphQLError_WithErrors(t *testing.T) {
	body := []byte(`{"errors":[{"message":"bad query"}]}`)
	err := graphQLError(body, "TestOp")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "TestOp") {
		t.Errorf("error should contain operation name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "bad query") {
		t.Errorf("error should contain message, got: %v", err)
	}
}

func TestGraphQLError_NoErrors(t *testing.T) {
	body := []byte(`{"data":{}}`)
	err := graphQLError(body, "TestOp")
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
}

func TestGraphqlURL_format(t *testing.T) {
	tests := []struct {
		op      string
		queryID string
		want    string
	}{
		{"CreateTweet", "abc123", GraphQLBaseURL + "/abc123/CreateTweet"},
		{"SearchTimeline", "xyz", GraphQLBaseURL + "/xyz/SearchTimeline"},
		{"", "", GraphQLBaseURL + "//"},
	}
	for _, tt := range tests {
		got := graphqlURL(tt.op, tt.queryID)
		if got != tt.want {
			t.Errorf("graphqlURL(%q, %q) = %q, want %q", tt.op, tt.queryID, got, tt.want)
		}
	}
}

func TestDoPOSTJSON_ContextCancelled(t *testing.T) {
	srv := testutil.NewTestServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := newBareTestClient(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.doPOSTJSON(ctx, srv.URL+"/post", c.getJSONHeaders(), map[string]string{"k": "v"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestRetryDelay_RetryAfterCappedAt60(t *testing.T) {
	got := retryDelay(0, "120")
	if got != 60*time.Second {
		t.Errorf("retryDelay(0, \"120\") = %v, want %v", got, 60*time.Second)
	}
}

func TestRetryDelay_RetryAfterExactly60(t *testing.T) {
	got := retryDelay(0, "60")
	if got != 60*time.Second {
		t.Errorf("retryDelay(0, \"60\") = %v, want %v", got, 60*time.Second)
	}
}

func TestRetryDelay_RetryAfterZeroFallsBackToBackoff(t *testing.T) {
	got := retryDelay(0, "0")
	if got >= time.Second {
		t.Errorf("retryDelay(0, \"0\") = %v, want < 1s (backoff range)", got)
	}
}

func TestRetryDelay_RetryAfterNegative(t *testing.T) {
	got := retryDelay(0, "-5")
	if got >= time.Second {
		t.Errorf("retryDelay(0, \"-5\") = %v, want < 1s (backoff range, not negative)", got)
	}
}
