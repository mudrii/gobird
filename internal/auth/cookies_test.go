package auth

import (
	"context"
	"testing"

	"github.com/mudrii/gobird/internal/types"
)

func TestPreferredDomainCookies_XComWins(t *testing.T) {
	cookies := []domainCookie{
		{domain: ".twitter.com", name: "auth_token", value: "tw_auth"},
		{domain: ".twitter.com", name: "ct0", value: "tw_ct0"},
		{domain: ".x.com", name: "auth_token", value: "x_auth"},
		{domain: ".x.com", name: "ct0", value: "x_ct0"},
	}
	authToken, ct0 := preferredDomainCookies(cookies)
	if authToken != "x_auth" {
		t.Errorf("authToken: want %q, got %q", "x_auth", authToken)
	}
	if ct0 != "x_ct0" {
		t.Errorf("ct0: want %q, got %q", "x_ct0", ct0)
	}
}

func TestPreferredDomainCookies_TwitterComFallback(t *testing.T) {
	cookies := []domainCookie{
		{domain: ".twitter.com", name: "auth_token", value: "tw_auth"},
		{domain: ".twitter.com", name: "ct0", value: "tw_ct0"},
	}
	authToken, ct0 := preferredDomainCookies(cookies)
	if authToken != "tw_auth" {
		t.Errorf("authToken: want %q, got %q", "tw_auth", authToken)
	}
	if ct0 != "tw_ct0" {
		t.Errorf("ct0: want %q, got %q", "tw_ct0", ct0)
	}
}

func TestPreferredDomainCookies_UnknownDomain(t *testing.T) {
	cookies := []domainCookie{
		{domain: ".example.com", name: "auth_token", value: "other_auth"},
		{domain: ".example.com", name: "ct0", value: "other_ct0"},
	}
	authToken, ct0 := preferredDomainCookies(cookies)
	if authToken != "other_auth" {
		t.Errorf("authToken: want %q, got %q", "other_auth", authToken)
	}
	if ct0 != "other_ct0" {
		t.Errorf("ct0: want %q, got %q", "other_ct0", ct0)
	}
}

func TestPreferredDomainCookies_EmptySlice(t *testing.T) {
	authToken, ct0 := preferredDomainCookies(nil)
	if authToken != "" {
		t.Errorf("authToken: want empty, got %q", authToken)
	}
	if ct0 != "" {
		t.Errorf("ct0: want empty, got %q", ct0)
	}
}

func TestPreferredDomainCookies_MixedDomains(t *testing.T) {
	cookies := []domainCookie{
		{domain: ".example.com", name: "auth_token", value: "other_auth"},
		{domain: ".twitter.com", name: "auth_token", value: "tw_auth"},
		{domain: ".example.com", name: "ct0", value: "other_ct0"},
		{domain: ".twitter.com", name: "ct0", value: "tw_ct0"},
	}
	authToken, ct0 := preferredDomainCookies(cookies)
	if authToken != "tw_auth" {
		t.Errorf("authToken: want %q, got %q", "tw_auth", authToken)
	}
	if ct0 != "tw_ct0" {
		t.Errorf("ct0: want %q, got %q", "tw_ct0", ct0)
	}
}

func TestPreferredDomainCookies_DotPrefixStripped(t *testing.T) {
	cookies := []domainCookie{
		{domain: "x.com", name: "auth_token", value: "x_nodot"},
		{domain: "x.com", name: "ct0", value: "x_ct0_nodot"},
	}
	authToken, ct0 := preferredDomainCookies(cookies)
	if authToken != "x_nodot" {
		t.Errorf("authToken: want %q, got %q", "x_nodot", authToken)
	}
	if ct0 != "x_ct0_nodot" {
		t.Errorf("ct0: want %q, got %q", "x_ct0_nodot", ct0)
	}
}

func TestPreferredDomainCookies_PartialCookies(t *testing.T) {
	cookies := []domainCookie{
		{domain: ".x.com", name: "auth_token", value: "x_auth"},
	}
	authToken, ct0 := preferredDomainCookies(cookies)
	if authToken != "x_auth" {
		t.Errorf("authToken: want %q, got %q", "x_auth", authToken)
	}
	if ct0 != "" {
		t.Errorf("ct0: want empty, got %q", ct0)
	}
}

func TestExtractFromBrowserOrder_EmptyOrder(t *testing.T) {
	prevSafari := safariExtractor
	prevChrome := chromeExtractor
	prevFirefox := firefoxExtractor
	safariExtractor = func(ctx context.Context) (*types.TwitterCookies, error) { return nil, context.DeadlineExceeded }
	chromeExtractor = func(ctx context.Context, profileHint string) (*types.TwitterCookies, error) {
		return nil, context.DeadlineExceeded
	}
	firefoxExtractor = func(ctx context.Context, profileHint string) (*types.TwitterCookies, error) {
		return nil, context.DeadlineExceeded
	}
	t.Cleanup(func() {
		safariExtractor = prevSafari
		chromeExtractor = prevChrome
		firefoxExtractor = prevFirefox
	})

	_, err := extractFromBrowserOrder(nil, ResolveOptions{CookieTimeoutMs: 1})
	if err == nil {
		t.Fatal("expected error when no browser finds cookies")
	}
}

func TestExtractFromBrowserOrder_UnknownBrowser(t *testing.T) {
	_, err := extractFromBrowserOrder([]string{"netscape"}, ResolveOptions{})
	if err == nil {
		t.Fatal("expected error for unknown browser")
	}
}

func TestBuildCookieHeader(t *testing.T) {
	cases := []struct {
		auth, ct0, want string
	}{
		{"tok1", "ct1", "auth_token=tok1; ct0=ct1"},
		{"", "", "auth_token=; ct0="},
		{"abc", "xyz", "auth_token=abc; ct0=xyz"},
	}
	for _, tc := range cases {
		got := buildCookieHeader(tc.auth, tc.ct0)
		if got != tc.want {
			t.Errorf("buildCookieHeader(%q, %q) = %q, want %q", tc.auth, tc.ct0, got, tc.want)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	cases := []struct {
		vals []string
		want string
	}{
		{[]string{"a", "b"}, "a"},
		{[]string{"", "b"}, "b"},
		{[]string{"", "", "c"}, "c"},
		{[]string{"", ""}, ""},
		{nil, ""},
	}
	for _, tc := range cases {
		got := firstNonEmpty(tc.vals...)
		if got != tc.want {
			t.Errorf("firstNonEmpty(%v) = %q, want %q", tc.vals, got, tc.want)
		}
	}
}

func TestValidateCredentials(t *testing.T) {
	cases := []struct {
		name      string
		authToken string
		ct0       string
		wantErr   bool
	}{
		{"valid", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", "abcdef1234567890abcdef1234567890ab", false},
		{"bad auth_token length", "short", "abcdef1234567890abcdef1234567890ab", true},
		{"auth_token with uppercase", "A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2", "abcdef1234567890abcdef1234567890ab", true},
		{"bad ct0 too short", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", "short", true},
		{"ct0 with special chars", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", "abcdef!234567890abcdef1234567890ab", true},
		{"both empty", "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCredentials(tc.authToken, tc.ct0)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateCredentials(%q, %q): err=%v, wantErr=%v", tc.authToken, tc.ct0, err, tc.wantErr)
			}
		})
	}
}
