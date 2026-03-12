package client

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Client is the single Twitter/X API client struct.
// All API methods are implemented as methods on this type, split across domain files.
type Client struct {
	authToken string
	ct0       string

	httpClient *http.Client

	// queryIDMu guards queryIDCache.
	queryIDMu    sync.RWMutex
	queryIDCache map[string]string
	// queryIDRefreshAt is when the cache was last refreshed.
	queryIDRefreshAt time.Time

	// userID is the authenticated user's numeric ID, resolved lazily.
	userIDOnce sync.Once
	userID     string
	userIDErr  error
}

// Options configures a Client at construction time.
type Options struct {
	// HTTPClient overrides the default http.Client (useful for testing).
	HTTPClient *http.Client
	// QueryIDCache seeds the runtime query ID cache (useful for testing).
	QueryIDCache map[string]string
}

// New creates a new Client with the given credentials.
// authToken and ct0 must not be empty.
func New(authToken, ct0 string, opts *Options) *Client {
	c := &Client{
		authToken:    authToken,
		ct0:          ct0,
		queryIDCache: make(map[string]string),
	}
	if opts != nil {
		if opts.HTTPClient != nil {
			c.httpClient = opts.HTTPClient
		}
		for k, v := range opts.QueryIDCache {
			c.queryIDCache[k] = v
		}
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return c
}

// getJsonHeaders returns JSON request headers for the authenticated user.
// Correction #70: getHeaders() = getJsonHeaders().
func (c *Client) getJsonHeaders() http.Header {
	return jsonHeaders(c.authToken, c.ct0)
}

// getBaseHeaders returns the base request headers without content-type.
func (c *Client) getBaseHeaders() http.Header {
	return baseHeaders(c.authToken, c.ct0)
}

// getUploadHeaders returns headers for media upload requests.
// Correction #70: upload uses base headers only.
func (c *Client) getUploadHeaders() http.Header {
	return uploadHeaders(c.authToken, c.ct0)
}

// ensureClientUserID resolves and caches the authenticated user's numeric ID.
// Called lazily before operations that require it (lists, etc.).
func (c *Client) ensureClientUserID(ctx context.Context) (string, error) {
	c.userIDOnce.Do(func() {
		u, err := c.getCurrentUser(ctx)
		if err != nil {
			c.userIDErr = err
			return
		}
		c.userID = u.ID
	})
	return c.userID, c.userIDErr
}
