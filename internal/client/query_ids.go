package client

import (
	"context"
	"io"
	"maps"
	"net/http"
	"regexp"
	"time"
)

// queryIDFormatRe validates scraped query IDs: 20+ alphanumeric/dash/underscore chars.
var queryIDFormatRe = regexp.MustCompile(`^[A-Za-z0-9_-]{20,}$`)

func (c *Client) scrapeBody(ctx context.Context, url string) ([]byte, error) {
	if err := c.waitForRateLimit(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("user-agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	closeErr := resp.Body.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &httpError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	return body, nil
}

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
// the in-memory cache. Errors are logged via c.logger but never returned, to
// preserve availability. Concurrent callers are coalesced via singleflight.
func (c *Client) refreshQueryIDs(ctx context.Context) {
	_, _, _ = c.refreshSF.Do("refresh", func() (any, error) {
		var refreshed map[string]string
		if c.scraper != nil {
			refreshed = c.scraper(ctx)
		} else {
			refreshed = c.scrapeQueryIDs(ctx)
		}

		c.queryIDMu.Lock()
		defer c.queryIDMu.Unlock()

		merged := make(map[string]string, len(BundledBaselineQueryIDs)+len(c.queryIDCache)+len(refreshed))
		maps.Copy(merged, BundledBaselineQueryIDs)
		maps.Copy(merged, c.queryIDCache)
		added := 0
		for op, id := range refreshed {
			if id != "" && queryIDFormatRe.MatchString(id) {
				merged[op] = id
				added++
			}
		}

		c.queryIDCache = merged
		c.queryIDRefreshAt = time.Now()
		if added == 0 {
			c.logger.WarnContext(ctx, "refreshQueryIDs produced no new query IDs",
				"scraped", len(refreshed))
		} else {
			c.logger.DebugContext(ctx, "refreshQueryIDs updated cache",
				"added", added, "scraped", len(refreshed))
		}
		return nil, nil
	})
}

// RefreshQueryIDs refreshes runtime query IDs from the X.com bundle.
func (c *Client) RefreshQueryIDs(ctx context.Context) {
	c.refreshQueryIDs(ctx)
}

func (c *Client) scrapeQueryIDs(ctx context.Context) map[string]string {
	// Cap total scrape time so a hanging upstream can't pin the refresh forever.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pages := []string{
		"https://x.com/home",
		"https://x.com/i/bookmarks",
		"https://x.com/explore",
		"https://x.com/settings/account",
	}
	scriptRe := regexp.MustCompile(`https://abs\.twimg\.com/[^"' )]+\.js`)

	// Precompile per-operation regexes once before the page loop to avoid
	// recompiling inside the inner loop for every JS bundle.
	operationREs := make(map[string]*regexp.Regexp, len(FallbackQueryIDs))
	for operation := range FallbackQueryIDs {
		operationREs[operation] = regexp.MustCompile(`([A-Za-z0-9_-]{20,})/` + regexp.QuoteMeta(operation) + `\b`)
	}

	found := map[string]string{}
	visitedScripts := map[string]struct{}{}

	for _, pageURL := range pages {
		if ctx.Err() != nil {
			break
		}
		body, err := c.scrapeBody(ctx, pageURL)
		if err != nil {
			c.logger.DebugContext(ctx, "scrape page failed", "url", pageURL, "err", err)
			continue
		}
		for _, scriptURL := range scriptRe.FindAllString(string(body), -1) {
			if _, dup := visitedScripts[scriptURL]; dup {
				continue
			}
			visitedScripts[scriptURL] = struct{}{}

			if ctx.Err() != nil {
				return found
			}
			scriptBody, err := c.scrapeBody(ctx, scriptURL)
			if err != nil {
				c.logger.DebugContext(ctx, "scrape script failed", "url", scriptURL, "err", err)
				continue
			}
			script := string(scriptBody)
			for operation, opRe := range operationREs {
				if _, ok := found[operation]; ok {
					continue
				}
				if m := opRe.FindStringSubmatch(script); len(m) > 1 {
					found[operation] = m[1]
				}
			}
			// Early exit once every known operation has a fresh ID.
			if len(found) >= len(operationREs) {
				return found
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
