// Package auth handles credential resolution from CLI flags, env vars, and browser cookies.
package auth

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mudrii/gobird/internal/types"
)

// ResolveOptions carries the candidate credentials from each source tier.
type ResolveOptions struct {
	// FlagAuthToken is set from --auth-token CLI flag.
	FlagAuthToken string
	// FlagCt0 is set from --ct0 CLI flag.
	FlagCt0 string
	// Browser names which browser to try ("safari", "chrome", "firefox", "").
	// Empty means try all in order.
	Browser string
	// CookieSources is the browser extraction order.
	CookieSources []string
	// ChromeProfile is the Chrome profile name or directory hint.
	ChromeProfile string
	// FirefoxProfile is the Firefox profile name hint.
	FirefoxProfile string
	// CookieTimeoutMs aborts browser cookie extraction when positive.
	CookieTimeoutMs int
}

// ResolveCredentials resolves auth_token and ct0 in priority order:
//  1. CLI flags (FlagAuthToken / FlagCt0)
//  2. Environment variables (AUTH_TOKEN > TWITTER_AUTH_TOKEN; CT0 > TWITTER_CT0)
//  3. Browser cookie extraction
//
// Returns an error if no valid credentials are found.
func ResolveCredentials(opts ResolveOptions) (*types.TwitterCookies, error) {
	// 1. CLI flags — both must be set for this tier to win.
	if opts.FlagAuthToken != "" && opts.FlagCt0 != "" {
		return &types.TwitterCookies{
			AuthToken:    opts.FlagAuthToken,
			Ct0:          opts.FlagCt0,
			CookieHeader: buildCookieHeader(opts.FlagAuthToken, opts.FlagCt0),
		}, nil
	}

	// 2. Environment variables.
	envToken := firstNonEmpty(os.Getenv("AUTH_TOKEN"), os.Getenv("TWITTER_AUTH_TOKEN"))
	envCt0 := firstNonEmpty(os.Getenv("CT0"), os.Getenv("TWITTER_CT0"))
	if envToken != "" && envCt0 != "" {
		return &types.TwitterCookies{
			AuthToken:    envToken,
			Ct0:          envCt0,
			CookieHeader: buildCookieHeader(envToken, envCt0),
		}, nil
	}

	// 3. Browser cookie extraction.
	order, err := resolveCookieSourceOrder(opts)
	if err != nil {
		return nil, err
	}
	creds, err := extractFromBrowserOrder(order, opts)
	if err != nil {
		return nil, fmt.Errorf("credential resolution failed: no valid credentials found (browser: %w)", err)
	}
	return creds, nil
}

func resolveCookieSourceOrder(opts ResolveOptions) ([]string, error) {
	if len(opts.CookieSources) > 0 {
		return normalizeCookieSources(opts.CookieSources)
	}
	if opts.Browser != "" {
		return normalizeCookieSources([]string{opts.Browser})
	}
	return []string{"safari", "chrome", "firefox"}, nil
}

func normalizeCookieSources(sources []string) ([]string, error) {
	out := make([]string, 0, len(sources))
	for _, source := range sources {
		source = strings.ToLower(strings.TrimSpace(source))
		switch source {
		case "safari", "chrome", "firefox":
			out = append(out, source)
		default:
			return nil, fmt.Errorf("invalid cookie source %q", source)
		}
	}
	return out, nil
}

func extractWithTimeout(timeoutMs int, fn func() (*types.TwitterCookies, error)) (*types.TwitterCookies, error) {
	if timeoutMs <= 0 {
		return fn()
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	type result struct {
		creds *types.TwitterCookies
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		creds, err := fn()
		ch <- result{creds: creds, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("cookie extraction timed out after %dms", timeoutMs)
	case res := <-ch:
		return res.creds, res.err
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func buildCookieHeader(authToken, ct0 string) string {
	return "auth_token=" + authToken + "; ct0=" + ct0
}

// ExtractSafariCookies reads Twitter cookies from Safari.
func ExtractSafariCookies() (*types.TwitterCookies, error) {
	return extractSafari()
}

// ExtractChromeCookies reads Twitter cookies from Chrome or Chromium.
func ExtractChromeCookies(profileHint string) (*types.TwitterCookies, error) {
	return extractChrome(profileHint)
}

// ExtractFirefoxCookies reads Twitter cookies from Firefox.
func ExtractFirefoxCookies(profileHint string) (*types.TwitterCookies, error) {
	return extractFirefox(profileHint)
}
