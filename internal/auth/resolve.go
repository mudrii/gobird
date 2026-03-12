// Package auth handles credential resolution from CLI flags, env vars, and browser cookies.
package auth

import (
	"fmt"
	"os"

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
	creds, err := extractFromBrowser(opts.Browser)
	if err != nil {
		return nil, fmt.Errorf("credential resolution failed: no valid credentials found (browser: %w)", err)
	}
	return creds, nil
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
