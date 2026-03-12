package client

import (
	"context"
	"time"
)

const queryIDCacheTTL = 24 * time.Hour

// getQueryID returns the active query ID for the given operation.
// Priority: runtime cache → bundled baseline → fallback.
func (c *Client) getQueryID(operation string) string {
	c.queryIDMu.RLock()
	id, ok := c.queryIDCache[operation]
	c.queryIDMu.RUnlock()
	if ok && id != "" {
		return id
	}
	if id, ok := BundledBaselineQueryIDs[operation]; ok {
		return id
	}
	return FallbackQueryIDs[operation]
}

// getQueryIDs returns all query IDs to try for an operation, in priority order.
// The first element is the runtime-cached or bundled primary; remaining are
// additional hardcoded fallbacks from PerOperationFallbackIDs.
func (c *Client) getQueryIDs(operation string) []string {
	primary := c.getQueryID(operation)

	all, ok := PerOperationFallbackIDs[operation]
	if !ok {
		if primary != "" {
			return []string{primary}
		}
		return nil
	}

	// Deduplicate: primary first, then the rest.
	seen := map[string]bool{primary: true}
	result := []string{primary}
	for _, id := range all {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}

// refreshQueryIDs scrapes fresh query IDs from the X.com bundle and updates
// the in-memory cache. Errors are silently ignored to preserve availability.
func (c *Client) refreshQueryIDs(ctx context.Context) {
	// TODO(Phase 3): implement bundle scraping.
	// For now, seeds the cache from the bundled baseline.
	c.queryIDMu.Lock()
	defer c.queryIDMu.Unlock()
	for op, id := range BundledBaselineQueryIDs {
		c.queryIDCache[op] = id
	}
	c.queryIDRefreshAt = time.Now()
}

// attemptResult carries the outcome of a single query-ID attempt.
type attemptResult struct {
	body    []byte
	err     error
	had404  bool
	success bool
}

// withRefreshedQueryIDsOn404 calls attempt; if it got a 404, refreshes query
// IDs and calls attempt again. Returns whether a refresh occurred.
// Used by getFollowing, getFollowers, getUserAboutAccount.
func (c *Client) withRefreshedQueryIDsOn404(ctx context.Context, attempt func() attemptResult) (attemptResult, bool) {
	r := attempt()
	if r.success || !r.had404 {
		return r, false
	}
	c.refreshQueryIDs(ctx)
	return attempt(), true
}
