package auth

import (
	"fmt"
	"strings"

	"github.com/mudrii/gobird/internal/types"
)

// extractFromBrowser tries each supported browser in order.
// If browser is non-empty, only that browser is tried.
// Domain preference: x.com > twitter.com > first match. Correction: doc §auth.
func extractFromBrowser(browser string) (*types.TwitterCookies, error) {
	if strings.TrimSpace(browser) == "" {
		return extractFromBrowserOrder(nil, ResolveOptions{})
	}
	return extractFromBrowserOrder([]string{browser}, ResolveOptions{Browser: browser})
}

// extractFromBrowserOrder tries each supported browser in order.
func extractFromBrowserOrder(order []string, opts ResolveOptions) (*types.TwitterCookies, error) {
	type extractor struct {
		name string
		fn   func() (*types.TwitterCookies, error)
	}
	all := []extractor{
		{"safari", func() (*types.TwitterCookies, error) { return extractSafari() }},
		{"chrome", func() (*types.TwitterCookies, error) { return extractChrome(opts.ChromeProfile) }},
		{"firefox", func() (*types.TwitterCookies, error) { return extractFirefox(opts.FirefoxProfile) }},
	}
	if len(order) == 0 {
		order = []string{"safari", "chrome", "firefox"}
	}
	var lastErr error
	for _, wanted := range order {
		matched := false
		for _, e := range all {
			if e.name != wanted {
				continue
			}
			matched = true
			creds, err := extractWithTimeout(opts.CookieTimeoutMs, e.fn)
			if err == nil && creds != nil {
				return creds, nil
			}
			if err != nil {
				lastErr = err
			} else {
				lastErr = fmt.Errorf("%s: no Twitter cookies found", e.name)
			}
			break
		}
		if !matched {
			return nil, fmt.Errorf("unknown browser: %q", wanted)
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no Twitter cookies found in any browser")
}

// preferredDomainCookies selects auth_token and ct0 from a list of (domain, name, value)
// tuples, preferring x.com > twitter.com > first match.
func preferredDomainCookies(cookies []domainCookie) (authToken, ct0 string) {
	domainRank := func(d string) int {
		d = strings.TrimPrefix(d, ".")
		switch d {
		case "x.com":
			return 0
		case "twitter.com":
			return 1
		default:
			return 2
		}
	}

	bestRankToken, bestRankCt0 := 99, 99

	for _, c := range cookies {
		rank := domainRank(c.domain)
		if c.name == "auth_token" && rank < bestRankToken {
			authToken = c.value
			bestRankToken = rank
		}
		if c.name == "ct0" && rank < bestRankCt0 {
			ct0 = c.value
			bestRankCt0 = rank
		}
	}
	return
}

// domainCookie holds a single cookie row from a browser store.
type domainCookie struct {
	domain string
	name   string
	value  string
}
