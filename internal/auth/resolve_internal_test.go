package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mudrii/gobird/internal/types"
)

func TestResolveCredentials_RejectsMalformedBrowserCredentials(t *testing.T) {
	prevSafari := safariExtractor
	prevChrome := chromeExtractor
	prevFirefox := firefoxExtractor

	safariExtractor = func(ctx context.Context) (*types.TwitterCookies, error) {
		return &types.TwitterCookies{
			AuthToken:    "bad\nheader",
			Ct0:          "short",
			CookieHeader: "auth_token=bad\nheader; ct0=short",
		}, nil
	}
	chromeExtractor = func(ctx context.Context, profileHint string) (*types.TwitterCookies, error) {
		return nil, errors.New("should not be called")
	}
	firefoxExtractor = func(ctx context.Context, profileHint string) (*types.TwitterCookies, error) {
		return nil, errors.New("should not be called")
	}
	t.Cleanup(func() {
		safariExtractor = prevSafari
		chromeExtractor = prevChrome
		firefoxExtractor = prevFirefox
	})

	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("TWITTER_AUTH_TOKEN", "")
	t.Setenv("CT0", "")
	t.Setenv("TWITTER_CT0", "")

	_, err := ResolveCredentials(ResolveOptions{Browser: "safari"})
	if err == nil {
		t.Fatal("expected malformed browser credentials to fail validation")
	}
	if !strings.Contains(err.Error(), "invalid credentials from browser") {
		t.Fatalf("expected browser validation error, got %v", err)
	}
}
