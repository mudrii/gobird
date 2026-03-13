package bird

import (
	"github.com/mudrii/gobird/internal/auth"
	"github.com/mudrii/gobird/internal/types"
)

// ResolveOptions configures credential resolution.
type ResolveOptions = auth.ResolveOptions

// TwitterCookies holds resolved authentication credentials.
type TwitterCookies = types.TwitterCookies

// ResolveCredentials resolves auth_token and ct0 in priority order:
//  1. CLI flags (FlagAuthToken / FlagCt0)
//  2. Environment variables
//  3. Browser cookie extraction
func ResolveCredentials(opts ResolveOptions) (*TwitterCookies, error) {
	return auth.ResolveCredentials(opts)
}

// ExtractSafariCookies reads Twitter cookies from Safari.
func ExtractSafariCookies() (*TwitterCookies, error) {
	return auth.ExtractSafariCookies()
}

// ExtractChromeCookies reads Twitter cookies from Chrome or Chromium.
func ExtractChromeCookies(profileHint string) (*TwitterCookies, error) {
	return auth.ExtractChromeCookies(profileHint)
}

// ExtractFirefoxCookies reads Twitter cookies from Firefox.
func ExtractFirefoxCookies(profileHint string) (*TwitterCookies, error) {
	return auth.ExtractFirefoxCookies(profileHint)
}
