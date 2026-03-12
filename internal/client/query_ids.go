package client

import (
	"context"
	"io"
	"net/http"
	"regexp"
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

// ActiveQueryID returns the active query ID for the given operation.
func (c *Client) ActiveQueryID(operation string) string {
	return c.getQueryID(operation)
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

// AllQueryIDs returns all query IDs to try for the given operation.
func (c *Client) AllQueryIDs(operation string) []string {
	return c.getQueryIDs(operation)
}

// refreshQueryIDs scrapes fresh query IDs from the X.com bundle and updates
// the in-memory cache. Errors are silently ignored to preserve availability.
func (c *Client) refreshQueryIDs(ctx context.Context) {
	refreshed := scrapeQueryIDs(ctx)
	c.queryIDMu.Lock()
	defer c.queryIDMu.Unlock()
	for op, id := range BundledBaselineQueryIDs {
		c.queryIDCache[op] = id
	}
	for op, id := range refreshed {
		if id != "" {
			c.queryIDCache[op] = id
		}
	}
	c.queryIDRefreshAt = time.Now()
}

// RefreshQueryIDs refreshes runtime query IDs from the X.com bundle.
func (c *Client) RefreshQueryIDs(ctx context.Context) {
	c.refreshQueryIDs(ctx)
}

func scrapeQueryIDs(ctx context.Context) map[string]string {
	pages := []string{
		"https://x.com/home",
		"https://x.com/i/bookmarks",
		"https://x.com/explore",
		"https://x.com/settings/account",
	}
	scriptRe := regexp.MustCompile(`https://abs\.twimg\.com/[^"' )]+\.js`)
	found := map[string]string{}
	client := &http.Client{Timeout: 15 * time.Second}
	visitedScripts := map[string]bool{}

	for _, pageURL := range pages {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("user-agent", UserAgent)
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}
		for _, scriptURL := range scriptRe.FindAllString(string(body), -1) {
			if visitedScripts[scriptURL] {
				continue
			}
			visitedScripts[scriptURL] = true

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, scriptURL, nil)
			if err != nil {
				continue
			}
			req.Header.Set("user-agent", UserAgent)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			scriptBody, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				continue
			}
			script := string(scriptBody)
			for operation := range FallbackQueryIDs {
				if _, ok := found[operation]; ok {
					continue
				}
				re := regexp.MustCompile(`([A-Za-z0-9_-]{20,})/` + regexp.QuoteMeta(operation) + `\b`)
				if m := re.FindStringSubmatch(script); len(m) > 1 {
					found[operation] = m[1]
				}
			}
		}
	}
	return found
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
