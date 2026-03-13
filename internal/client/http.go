package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// httpError represents a non-2xx HTTP response.
type httpError struct {
	StatusCode int
	Body       string
}

func (e *httpError) Error() string {
	body := e.Body
	const maxBodyLen = 200
	if len(body) > maxBodyLen {
		body = body[:maxBodyLen] + "…"
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, body)
}

// is404 reports whether err is an HTTP 404 error.
func is404(err error) bool {
	var he *httpError
	if e, ok := err.(*httpError); ok {
		he = e
	}
	return he != nil && he.StatusCode == 404
}

// doGET performs a GET request with the given headers and returns the response body.
func (c *Client) doGET(ctx context.Context, url string, headers http.Header) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	return c.do(req)
}

// doPOSTJSON performs a POST request with a JSON-encoded body.
func (c *Client) doPOSTJSON(ctx context.Context, url string, headers http.Header, body any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	return c.do(req)
}

// doPOSTForm performs a POST request with an application/x-www-form-urlencoded body.
func (c *Client) doPOSTForm(ctx context.Context, url string, headers http.Header, body string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &httpError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	return body, nil
}

// retryableStatus reports whether an HTTP status code should be retried.
func retryableStatus(code int) bool {
	return code == 429 || code == 500 || code == 502 || code == 503 || code == 504
}

// fetchWithRetry performs an HTTP GET with exponential-backoff retry.
// Used ONLY for Bookmarks and BookmarkFolderTimeline.
// maxRetries=2 → 3 total attempts (0, 1, 2). Correction #86: post-loop return is dead code.
func (c *Client) fetchWithRetry(ctx context.Context, url string, headers http.Header) ([]byte, error) {
	const maxRetries = 2
	const baseDelayMs = 500

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		for k, vs := range headers {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close() //nolint:errcheck
		if readErr != nil {
			return nil, readErr
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, nil
		}

		httpErr := &httpError{StatusCode: resp.StatusCode, Body: string(body)}
		lastErr = httpErr

		if !retryableStatus(resp.StatusCode) || attempt == maxRetries {
			return nil, lastErr
		}

		var delay time.Duration
		if ra := resp.Header.Get("retry-after"); ra != "" {
			if secs, parseErr := strconv.Atoi(ra); parseErr == nil {
				delay = time.Duration(secs) * time.Second
			}
		}
		if delay == 0 {
			jitter := time.Duration(rand.Intn(baseDelayMs)) * time.Millisecond
			delay = time.Duration(baseDelayMs<<uint(attempt))*time.Millisecond + jitter
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
	// Correction #86: this return is dead code — the loop above always returns.
	return nil, lastErr
}

// graphqlURL builds a GraphQL endpoint URL for the given operation and query ID.
func graphqlURL(operation, queryID string) string {
	return fmt.Sprintf("%s/%s/%s", GraphQLBaseURL, queryID, operation)
}

// parseGraphQLErrors checks a JSON response body for GraphQL-level errors and
// returns them if present. Does not return an error for partial data.
func parseGraphQLErrors(body []byte) []graphqlError {
	var env struct {
		Errors []graphqlError `json:"errors"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil
	}
	return env.Errors
}

func graphQLError(body []byte, operation string) error {
	errs := parseGraphQLErrors(body)
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%s: %s", operation, errs[0].Message)
}

type graphqlError struct {
	Message    string          `json:"message"`
	Extensions errorExtensions `json:"extensions"`
	Path       []any           `json:"path"`
}

type errorExtensions struct {
	Code string `json:"code"`
}
