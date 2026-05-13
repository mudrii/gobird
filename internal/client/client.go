package client

import (
	"context"
	"io"
	"log/slog"
	"maps"
	mrand "math/rand/v2"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
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

	// rateMu guards nextRequestAt for the global rate limiter.
	rateMu sync.Mutex
	// nextRequestAt is the next reserved request slot.
	nextRequestAt time.Time
	// minInterval is the minimum duration between HTTP requests.
	minInterval time.Duration

	// scraper overrides scrapeQueryIDs for testing. If nil, the real scraper is used.
	scraper func(ctx context.Context) map[string]string

	// logger emits structured diagnostic events. Defaults to a discarding handler
	// so library users see nothing unless they opt in via Options.Logger.
	logger *slog.Logger

	// refreshSF coalesces concurrent refreshQueryIDs calls. A stampede of 404s
	// would otherwise trigger one scrape per goroutine.
	refreshSF singleflight.Group

	// userIDSF coalesces concurrent getCurrentUser calls during cold-start
	// user-ID resolution.
	userIDSF singleflight.Group
}

// Options configures a Client at construction time.
type Options struct {
	// HTTPClient overrides the default http.Client (useful for testing).
	HTTPClient *http.Client
	// QueryIDCache seeds the runtime query ID cache (useful for testing).
	QueryIDCache map[string]string
	// TimeoutMs overrides the default HTTP timeout when HTTPClient is not supplied.
	TimeoutMs int
	// RequestsPerSecond sets the global rate limit. Default: 1.0 (one request per second).
	// Set to 0 or negative to disable rate limiting.
	RequestsPerSecond float64
	// Logger receives structured diagnostic events (query-ID refresh failures,
	// scrape errors). When nil, events are discarded.
	Logger *slog.Logger
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
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	// Default rate limit: 1 request per second.
	rps := 1.0
	if opts != nil {
		if opts.HTTPClient != nil {
			c.httpClient = opts.HTTPClient
		}
		maps.Copy(c.queryIDCache, opts.QueryIDCache)
		if opts.RequestsPerSecond > 0 {
			rps = opts.RequestsPerSecond
		} else if opts.RequestsPerSecond < 0 {
			rps = 0
		}
		if opts.Logger != nil {
			c.logger = opts.Logger
		}
	}
	if rps > 0 {
		c.minInterval = time.Duration(float64(time.Second) / rps)
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
func (c *Client) getJSONHeaders() (http.Header, error) {
	return jsonHeaders(c.authToken, c.ct0, c.clientUUID, c.deviceID, c.cachedUserID())
}

// getBaseHeaders returns the base request headers without content-type.
func (c *Client) getBaseHeaders() (http.Header, error) {
	return baseHeaders(c.authToken, c.ct0, c.clientUUID, c.deviceID, c.cachedUserID())
}

// getUploadHeaders returns headers for media upload requests.
// Correction #70: upload uses base headers only.
func (c *Client) getUploadHeaders() (http.Header, error) {
	return uploadHeaders(c.authToken, c.ct0, c.clientUUID, c.deviceID, c.cachedUserID())
}

// waitForRateLimit reserves the next request slot and waits until it is due.
// Reservation happens under the mutex so concurrent callers cannot claim the
// same slot. The wait itself is context-aware and happens outside the lock.
func (c *Client) waitForRateLimit(ctx context.Context) error {
	if c.minInterval <= 0 {
		return nil
	}

	c.rateMu.Lock()
	waitUntil := time.Now()
	if c.nextRequestAt.After(waitUntil) {
		waitUntil = c.nextRequestAt
	}
	reservedNext := waitUntil.Add(c.minInterval)
	c.nextRequestAt = reservedNext
	c.rateMu.Unlock()

	if delay := time.Until(waitUntil); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			c.rateMu.Lock()
			if c.nextRequestAt.Equal(reservedNext) {
				c.nextRequestAt = waitUntil
			}
			c.rateMu.Unlock()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return nil
}

// paginationJitter returns a random duration between 50ms and 150ms.
func paginationJitter() time.Duration {
	// #nosec G404 -- jitter is for pacing only; cryptographic randomness not required.
	return time.Duration(50+mrand.IntN(101)) * time.Millisecond
}

// ensureClientUserID resolves and caches the authenticated user's numeric ID.
// Called lazily before operations that require it (lists, etc.).
// Only a successful result is cached; errors are not, so callers may retry.
// Concurrent callers are coalesced via singleflight so a cold start fires at
// most one getCurrentUser request.
func (c *Client) ensureClientUserID(ctx context.Context) error {
	// Fast path: already cached.
	c.userIDMu.RLock()
	id := c.userID
	c.userIDMu.RUnlock()
	if id != "" {
		return nil
	}

	_, err, _ := c.userIDSF.Do("currentUser", func() (any, error) {
		// Re-check under singleflight in case a sibling already populated the cache.
		c.userIDMu.RLock()
		if c.userID != "" {
			c.userIDMu.RUnlock()
			return nil, nil
		}
		c.userIDMu.RUnlock()

		user, err := c.getCurrentUser(ctx)
		if err != nil {
			return nil, err
		}
		c.userIDMu.Lock()
		if c.userID == "" {
			c.userID = user.ID
		}
		c.userIDMu.Unlock()
		return nil, nil
	})
	return err
}
