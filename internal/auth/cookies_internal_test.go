package auth

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/mudrii/gobird/internal/types"
)

func TestExtractFromBrowserOrder_UsesConfiguredOrder(t *testing.T) {
	prevSafari := safariExtractor
	prevChrome := chromeExtractor
	prevFirefox := firefoxExtractor
	var mu sync.Mutex
	callOrder := make([]string, 0, 2)

	safariExtractor = func(ctx context.Context) (*types.TwitterCookies, error) {
		mu.Lock()
		callOrder = append(callOrder, "safari")
		mu.Unlock()
		return nil, errors.New("miss")
	}
	chromeExtractor = func(ctx context.Context, profileHint string) (*types.TwitterCookies, error) {
		mu.Lock()
		callOrder = append(callOrder, "chrome:"+profileHint)
		mu.Unlock()
		return &types.TwitterCookies{AuthToken: "tok", Ct0: "ct0"}, nil
	}
	firefoxExtractor = func(ctx context.Context, profileHint string) (*types.TwitterCookies, error) {
		mu.Lock()
		callOrder = append(callOrder, "firefox:"+profileHint)
		mu.Unlock()
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
	mu.Lock()
	order := make([]string, len(callOrder))
	copy(order, callOrder)
	mu.Unlock()
	if len(order) != 2 || order[0] != "safari" || order[1] != "chrome:work" {
		t.Fatalf("unexpected call order: %v", order)
	}
}

func TestExtractFromBrowserOrder_TimeoutFallsThroughToNextBrowser(t *testing.T) {
	prevSafari := safariExtractor
	prevChrome := chromeExtractor
	prevFirefox := firefoxExtractor
	var mu sync.Mutex
	callOrder := make([]string, 0, 2)

	safariExtractor = func(ctx context.Context) (*types.TwitterCookies, error) {
		mu.Lock()
		callOrder = append(callOrder, "safari")
		mu.Unlock()
		<-ctx.Done()
		return nil, ctx.Err()
	}
	chromeExtractor = func(ctx context.Context, profileHint string) (*types.TwitterCookies, error) {
		mu.Lock()
		callOrder = append(callOrder, "chrome")
		mu.Unlock()
		return &types.TwitterCookies{AuthToken: "tok", Ct0: "ct0"}, nil
	}
	firefoxExtractor = func(ctx context.Context, profileHint string) (*types.TwitterCookies, error) {
		mu.Lock()
		callOrder = append(callOrder, "firefox")
		mu.Unlock()
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
	mu.Lock()
	order := make([]string, len(callOrder))
	copy(order, callOrder)
	mu.Unlock()
	if len(order) != 2 || order[0] != "safari" || order[1] != "chrome" {
		t.Fatalf("unexpected call order: %v", order)
	}
}
