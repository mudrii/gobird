// Package testutil provides shared test helpers.
package testutil

import (
	"net/http"
	"net/http/httptest"
)

// HandlerFunc wraps a function to satisfy http.Handler.
type HandlerFunc = http.HandlerFunc

// NewTestServer starts a test HTTP server with the given handler and returns the server.
// Call server.Close() when done.
func NewTestServer(h http.Handler) *httptest.Server {
	return httptest.NewServer(h)
}

// StaticHandler returns an http.Handler that always responds with the given
// status code and body.
func StaticHandler(code int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	})
}

// RoundTripFunc is an http.RoundTripper backed by a function.
type RoundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements http.RoundTripper.
func (f RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// NewHTTPClientForServer returns an *http.Client that redirects all requests
// to the given httptest.Server URL, preserving path and query string.
// This is useful for wiring a mock server into code that constructs its own
// HTTP client internally.
func NewHTTPClientForServer(srv *httptest.Server) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(func(r *http.Request) (*http.Response, error) {
			newURL := srv.URL + r.URL.RequestURI()
			newReq, err := http.NewRequestWithContext(r.Context(), r.Method, newURL, r.Body)
			if err != nil {
				return nil, err
			}
			for k, vs := range r.Header {
				newReq.Header[k] = vs
			}
			return http.DefaultTransport.RoundTrip(newReq)
		}),
	}
}
