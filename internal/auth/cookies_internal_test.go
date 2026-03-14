package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/mudrii/gobird/internal/types"
)

func TestExtractFromBrowserOrder_UsesConfiguredOrder(t *testing.T) {
	prevSafari := safariExtractor
	prevChrome := chromeExtractor
	prevFirefox := firefoxExtractor
	callOrder := make([]string, 0, 2)

	safariExtractor = func(ctx context.Context) (*types.TwitterCookies, error) {
		callOrder = append(callOrder, "safari")
		return nil, errors.New("miss")
	}
	chromeExtractor = func(ctx context.Context, profileHint string) (*types.TwitterCookies, error) {
		callOrder = append(callOrder, "chrome:"+profileHint)
		return &types.TwitterCookies{AuthToken: "tok", Ct0: "ct0"}, nil
	}
	firefoxExtractor = func(ctx context.Context, profileHint string) (*types.TwitterCookies, error) {
		callOrder = append(callOrder, "firefox:"+profileHint)
		return nil, errors.New("should not be called")
	}
	t.Cleanup(func() {
		safariExtractor = prevSafari
		chromeExtractor = prevChrome
		firefoxExtractor = prevFirefox
	})

	creds, err := extractFromBrowserOrder([]string{"safari", "chrome"}, ResolveOptions{
		ChromeProfile: "work",
	})
	if err != nil {
		t.Fatalf("extractFromBrowserOrder: %v", err)
	}
	if creds == nil || creds.AuthToken != "tok" || creds.Ct0 != "ct0" {
		t.Fatalf("unexpected creds: %+v", creds)
	}
	if len(callOrder) != 2 || callOrder[0] != "safari" || callOrder[1] != "chrome:work" {
		t.Fatalf("unexpected call order: %v", callOrder)
	}
}

func TestExtractFromBrowserOrder_TimeoutFallsThroughToNextBrowser(t *testing.T) {
	prevSafari := safariExtractor
	prevChrome := chromeExtractor
	prevFirefox := firefoxExtractor
	callOrder := make([]string, 0, 2)

	safariExtractor = func(ctx context.Context) (*types.TwitterCookies, error) {
		callOrder = append(callOrder, "safari")
		<-ctx.Done()
		return nil, ctx.Err()
	}
	chromeExtractor = func(ctx context.Context, profileHint string) (*types.TwitterCookies, error) {
		callOrder = append(callOrder, "chrome")
		return &types.TwitterCookies{AuthToken: "tok", Ct0: "ct0"}, nil
	}
	firefoxExtractor = func(ctx context.Context, profileHint string) (*types.TwitterCookies, error) {
		callOrder = append(callOrder, "firefox")
		return nil, errors.New("should not be called")
	}
	t.Cleanup(func() {
		safariExtractor = prevSafari
		chromeExtractor = prevChrome
		firefoxExtractor = prevFirefox
	})

	creds, err := extractFromBrowserOrder([]string{"safari", "chrome"}, ResolveOptions{
		CookieTimeoutMs: 1,
	})
	if err != nil {
		t.Fatalf("extractFromBrowserOrder: %v", err)
	}
	if creds == nil || creds.AuthToken != "tok" || creds.Ct0 != "ct0" {
		t.Fatalf("unexpected creds: %+v", creds)
	}
	if len(callOrder) != 2 || callOrder[0] != "safari" || callOrder[1] != "chrome" {
		t.Fatalf("unexpected call order: %v", callOrder)
	}
}
