package client

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Client is the single Twitter/X API client struct.
// All API methods are implemented as methods on this type, split across domain files.
type Client struct {
	authToken string
	ct0       string

	httpClient *http.Client

	clientUUID string
	deviceID   string

	// queryIDMu guards queryIDCache.
	queryIDMu    sync.RWMutex
	queryIDCache map[string]string
	// queryIDRefreshAt is when the cache was last refreshed.
	queryIDRefreshAt time.Time

	// userID is the authenticated user's numeric ID, resolved lazily.
	// userIDMu guards userID; only a successful resolution is cached.
	userIDMu sync.RWMutex
	userID   string

	// scraper overrides scrapeQueryIDs for testing. If nil, the real scraper is used.
	scraper func(ctx context.Context) map[string]string
}

// Options configures a Client at construction time.
type Options struct {
	// HTTPClient overrides the default http.Client (useful for testing).
	HTTPClient *http.Client
	// QueryIDCache seeds the runtime query ID cache (useful for testing).
	QueryIDCache map[string]string
	// TimeoutMs overrides the default HTTP timeout when HTTPClient is not supplied.
	TimeoutMs int
}

// New creates a new Client with the given credentials.
// Callers are expected to validate credentials before constructing the client.
func New(authToken, ct0 string, opts *Options) *Client {
	c := &Client{
		authToken:    authToken,
		ct0:          ct0,
		queryIDCache: make(map[string]string),
		clientUUID:   uuid.NewString(),
		deviceID:     uuid.NewString(),
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
		timeout := 30 * time.Second
		if opts != nil && opts.TimeoutMs > 0 {
			timeout = time.Duration(opts.TimeoutMs) * time.Millisecond
		}
		c.httpClient = &http.Client{Timeout: timeout}
	}
	return c
}

// cachedUserID returns the cached user ID under the read lock.
func (c *Client) cachedUserID() string {
	c.userIDMu.RLock()
	id := c.userID
	c.userIDMu.RUnlock()
	return id
}

// getJSONHeaders returns JSON request headers for the authenticated user.
// Correction #70: getHeaders() = getJSONHeaders().
func (c *Client) getJSONHeaders() http.Header {
	return jsonHeaders(c.authToken, c.ct0, c.clientUUID, c.deviceID, c.cachedUserID())
}

// getBaseHeaders returns the base request headers without content-type.
func (c *Client) getBaseHeaders() http.Header {
	return baseHeaders(c.authToken, c.ct0, c.clientUUID, c.deviceID, c.cachedUserID())
}

// getUploadHeaders returns headers for media upload requests.
// Correction #70: upload uses base headers only.
func (c *Client) getUploadHeaders() http.Header {
	return uploadHeaders(c.authToken, c.ct0, c.clientUUID, c.deviceID, c.cachedUserID())
}

// ensureClientUserID resolves and caches the authenticated user's numeric ID.
// Called lazily before operations that require it (lists, etc.).
// Only a successful result is cached; errors are not, so callers may retry.
func (c *Client) ensureClientUserID(ctx context.Context) error {
	// Fast path: already cached.
	c.userIDMu.RLock()
	id := c.userID
	c.userIDMu.RUnlock()
	if id != "" {
		return nil
	}

	// Slow path: resolve without holding any lock (getCurrentUser uses the read lock
	// internally via cachedUserID, so we must not hold the write lock during the call).
	user, err := c.getCurrentUser(ctx)
	if err != nil {
		return err
	}

	c.userIDMu.Lock()
	if c.userID == "" {
		c.userID = user.ID
	}
	c.userIDMu.Unlock()
	return nil
}
