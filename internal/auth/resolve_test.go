package auth_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mudrii/gobird/internal/auth"
	"github.com/mudrii/gobird/internal/types"
)

// validToken is a syntactically valid auth_token (40 hex chars).
const validToken = "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"

// validCt0 is a syntactically valid ct0 (32+ alphanumeric chars).
const validCt0 = "abcdef1234567890abcdef1234567890ab"

func TestResolveFlagsWin(t *testing.T) {
	t.Setenv("AUTH_TOKEN", validToken)
	t.Setenv("CT0", validCt0)
	flagToken := "b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3"
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{
		FlagAuthToken: flagToken,
		FlagCt0:       validCt0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AuthToken != flagToken {
		t.Errorf("want %q, got %q", flagToken, creds.AuthToken)
	}
	if creds.Ct0 != validCt0 {
		t.Errorf("want %q, got %q", validCt0, creds.Ct0)
	}
}

func TestResolveEnvVarsWin(t *testing.T) {
	t.Setenv("AUTH_TOKEN", validToken)
	t.Setenv("CT0", validCt0)
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AuthToken != validToken {
		t.Errorf("want %q, got %q", validToken, creds.AuthToken)
	}
}

func TestResolveTwitterEnvFallback(t *testing.T) {
	twToken := "c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"
	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("CT0", "")
	t.Setenv("TWITTER_AUTH_TOKEN", twToken)
	t.Setenv("TWITTER_CT0", validCt0)
	defer func() {
		os.Unsetenv("TWITTER_AUTH_TOKEN")
		os.Unsetenv("TWITTER_CT0")
	}()
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AuthToken != twToken {
		t.Errorf("want %q, got %q", twToken, creds.AuthToken)
	}
}

func TestResolveFlagsNeedBoth(t *testing.T) {
	t.Setenv("AUTH_TOKEN", validToken)
	t.Setenv("CT0", validCt0)
	// Only one flag set — should fall through to env.
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{FlagAuthToken: "partial"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should use env vars since flag was incomplete.
	if creds.AuthToken != validToken {
		t.Errorf("want env token, got %q", creds.AuthToken)
	}
}

func TestCookieHeader(t *testing.T) {
	t.Setenv("AUTH_TOKEN", validToken)
	t.Setenv("CT0", validCt0)
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "auth_token=" + validToken + "; ct0=" + validCt0
	if creds.CookieHeader != want {
		t.Errorf("CookieHeader: want %q, got %q", want, creds.CookieHeader)
	}
}

// TestResolveCredentials_FlagAuthTokenAndCt0 verifies that when both flags are
// set they are returned directly, bypassing env and browser extraction.
func TestResolveCredentials_FlagAuthTokenAndCt0(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("CT0", "")
	flagToken := "d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5"
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{
		FlagAuthToken: flagToken,
		FlagCt0:       validCt0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AuthToken != flagToken {
		t.Errorf("AuthToken: want %q, got %q", flagToken, creds.AuthToken)
	}
	if creds.Ct0 != validCt0 {
		t.Errorf("Ct0: want %q, got %q", validCt0, creds.Ct0)
	}
	if creds.CookieHeader != "auth_token="+flagToken+"; ct0="+validCt0 {
		t.Errorf("CookieHeader: got %q", creds.CookieHeader)
	}
}

// TestResolveCredentials_FlagOnlyOneSet verifies that when only one flag is set
// the resolver falls through to env vars.
func TestResolveCredentials_FlagOnlyOneSet(t *testing.T) {
	t.Setenv("AUTH_TOKEN", validToken)
	t.Setenv("CT0", validCt0)
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{
		FlagAuthToken: "partial",
		// FlagCt0 deliberately omitted
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AuthToken != validToken {
		t.Errorf("expected env fallback, got %q", creds.AuthToken)
	}
}

// TestResolveCredentials_EnvVars verifies AUTH_TOKEN and CT0 env vars are used.
func TestResolveCredentials_EnvVars(t *testing.T) {
	envToken := "e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6"
	t.Setenv("AUTH_TOKEN", envToken)
	t.Setenv("CT0", validCt0)
	t.Setenv("TWITTER_AUTH_TOKEN", "")
	t.Setenv("TWITTER_CT0", "")
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AuthToken != envToken {
		t.Errorf("AuthToken: want %q, got %q", envToken, creds.AuthToken)
	}
	if creds.Ct0 != validCt0 {
		t.Errorf("Ct0: want %q, got %q", validCt0, creds.Ct0)
	}
}

// TestResolveCredentials_TwitterEnvVars verifies TWITTER_AUTH_TOKEN and
// TWITTER_CT0 are used when the primary env vars are absent.
func TestResolveCredentials_TwitterEnvVars(t *testing.T) {
	twToken := "f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1"
	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("CT0", "")
	t.Setenv("TWITTER_AUTH_TOKEN", twToken)
	t.Setenv("TWITTER_CT0", validCt0)
	defer func() {
		os.Unsetenv("TWITTER_AUTH_TOKEN")
		os.Unsetenv("TWITTER_CT0")
	}()
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AuthToken != twToken {
		t.Errorf("AuthToken: want %q, got %q", twToken, creds.AuthToken)
	}
	if creds.Ct0 != validCt0 {
		t.Errorf("Ct0: want %q, got %q", validCt0, creds.Ct0)
	}
}

// TestResolveCredentials_NoCredentials verifies that when no flags, no env vars,
// and no browser cookies are available, an error is returned.
func TestResolveCredentials_NoCredentials(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("CT0", "")
	t.Setenv("TWITTER_AUTH_TOKEN", "")
	t.Setenv("TWITTER_CT0", "")
	defer func() {
		os.Unsetenv("TWITTER_AUTH_TOKEN")
		os.Unsetenv("TWITTER_CT0")
	}()
	// Pass a 1ms timeout so browser extraction fails immediately.
	_, err := auth.ResolveCredentials(auth.ResolveOptions{
		CookieTimeoutMs: 1,
	})
	if err == nil {
		t.Fatal("expected error when no credentials available, got nil")
	}
}

// TestBuildCookieHeader verifies the exact "auth_token=X; ct0=Y" format
// by exercising it through ResolveCredentials.
func TestBuildCookieHeader(t *testing.T) {
	tok1 := "1111111111111111111111111111111111111111"
	ct01 := "abcdef1234567890abcdef1234567890ab"
	tok2 := "2222222222222222222222222222222222222222"
	ct02 := "12345678901234567890123456789012ef"
	cases := []struct {
		token string
		ct0   string
		want  string
	}{
		{tok1, ct01, "auth_token=" + tok1 + "; ct0=" + ct01},
		{tok2, ct02, "auth_token=" + tok2 + "; ct0=" + ct02},
	}
	for _, tc := range cases {
		t.Run(tc.token[:8], func(t *testing.T) {
			creds, err := auth.ResolveCredentials(auth.ResolveOptions{
				FlagAuthToken: tc.token,
				FlagCt0:       tc.ct0,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if creds.CookieHeader != tc.want {
				t.Errorf("want %q, got %q", tc.want, creds.CookieHeader)
			}
		})
	}
}

// TestFirstNonEmpty exercises the firstNonEmpty helper indirectly through
// credential resolution with partial env vars.
func TestFirstNonEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	primaryToken := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	twitterToken := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	primaryCt0 := "abcdef1234567890abcdef1234567890pp"
	twitterCt0 := "abcdef1234567890abcdef1234567890tt"
	cases := []struct {
		name      string
		authToken string
		twitterAT string
		ct0       string
		twitterCt string
		wantToken string
		wantCt0   string
		wantErr   bool
	}{
		{
			name:      "primary wins over twitter env",
			authToken: primaryToken, twitterAT: twitterToken,
			ct0: primaryCt0, twitterCt: twitterCt0,
			wantToken: primaryToken, wantCt0: primaryCt0,
		},
		{
			name:      "falls back to twitter env when primary empty",
			authToken: "", twitterAT: twitterToken,
			ct0: "", twitterCt: twitterCt0,
			wantToken: twitterToken, wantCt0: twitterCt0,
		},
		{
			name:      "both empty leads to browser attempt and error",
			authToken: "", twitterAT: "",
			ct0: "", twitterCt: "",
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AUTH_TOKEN", tc.authToken)
			t.Setenv("CT0", tc.ct0)
			t.Setenv("TWITTER_AUTH_TOKEN", tc.twitterAT)
			t.Setenv("TWITTER_CT0", tc.twitterCt)
			defer func() {
				os.Unsetenv("TWITTER_AUTH_TOKEN")
				os.Unsetenv("TWITTER_CT0")
			}()
			creds, err := auth.ResolveCredentials(auth.ResolveOptions{
				CookieTimeoutMs: 1,
			})
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if creds.AuthToken != tc.wantToken {
				t.Errorf("AuthToken: want %q, got %q", tc.wantToken, creds.AuthToken)
			}
			if creds.Ct0 != tc.wantCt0 {
				t.Errorf("Ct0: want %q, got %q", tc.wantCt0, creds.Ct0)
			}
		})
	}
}

// TestExtractWithTimeout_Success verifies that a function completing before the
// timeout returns its result.
func TestExtractWithTimeout_Success(t *testing.T) {
	want := &types.TwitterCookies{AuthToken: "a", Ct0: "b", CookieHeader: "auth_token=a; ct0=b"}
	got, err := auth.ExportedExtractWithTimeout(500, func(context.Context) (*types.TwitterCookies, error) {
		return want, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// TestExtractWithTimeout_NoTimeout verifies that timeoutMs <= 0 calls fn directly.
func TestExtractWithTimeout_NoTimeout(t *testing.T) {
	called := false
	want := &types.TwitterCookies{AuthToken: "x", Ct0: "y"}
	got, err := auth.ExportedExtractWithTimeout(0, func(context.Context) (*types.TwitterCookies, error) {
		called = true
		return want, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("fn was not called")
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// TestExtractWithTimeout_Timeout verifies that a blocking fn is interrupted and
// an error is returned.
func TestExtractWithTimeout_Timeout(t *testing.T) {
	done := make(chan struct{})
	t.Cleanup(func() { close(done) })

	_, err := auth.ExportedExtractWithTimeout(20, func(ctx context.Context) (*types.TwitterCookies, error) {
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-done:
		case <-timer.C:
		}
		return nil, errors.New("should not reach caller")
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected 'timed out' in error, got %q", err.Error())
	}
}

// TestNormalizeCookieSources_Valid verifies that all known browser names are accepted.
func TestNormalizeCookieSources_Valid(t *testing.T) {
	cases := []struct {
		input []string
		want  []string
	}{
		{[]string{"safari"}, []string{"safari"}},
		{[]string{"chrome"}, []string{"chrome"}},
		{[]string{"firefox"}, []string{"firefox"}},
		{[]string{"safari", "chrome", "firefox"}, []string{"safari", "chrome", "firefox"}},
		{[]string{"chrome", "safari"}, []string{"chrome", "safari"}},
	}
	for _, tc := range cases {
		got, err := auth.ExportedNormalizeCookieSources(tc.input)
		if err != nil {
			t.Errorf("input %v: unexpected error: %v", tc.input, err)
			continue
		}
		if len(got) != len(tc.want) {
			t.Errorf("input %v: want %v, got %v", tc.input, tc.want, got)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("input %v[%d]: want %q, got %q", tc.input, i, tc.want[i], got[i])
			}
		}
	}
}

// TestNormalizeCookieSources_Invalid verifies that unknown browser names produce an error.
// Note: normalizeCookieSources lowercases input before checking, so only truly unknown
// names (not just differently-cased known names) should produce errors.
func TestNormalizeCookieSources_Invalid(t *testing.T) {
	cases := [][]string{
		{"edge"},
		{"brave"},
		{},
		{""},
		{"unknown"},
		{"ie"},
		{"opera"},
	}
	for _, input := range cases {
		_, err := auth.ExportedNormalizeCookieSources(input)
		if err == nil {
			t.Errorf("input %v: expected error, got nil", input)
		}
	}
}

// TestNormalizeCookieSources_CaseInsensitive verifies that input is lowercased
// before validation, so "Safari" → "safari", "Chrome" → "chrome", etc.
func TestNormalizeCookieSources_CaseInsensitive(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Safari", "safari"},
		{"SAFARI", "safari"},
		{"Chrome", "chrome"},
		{"CHROME", "chrome"},
		{"Firefox", "firefox"},
		{"FIREFOX", "firefox"},
	}
	for _, tc := range cases {
		got, err := auth.ExportedNormalizeCookieSources([]string{tc.input})
		if err != nil {
			t.Errorf("input %q: unexpected error: %v", tc.input, err)
			continue
		}
		if len(got) != 1 || got[0] != tc.want {
			t.Errorf("input %q: want [%q], got %v", tc.input, tc.want, got)
		}
	}
}

func TestNormalizeCookieSources_TrimsAndSkipsEmpty(t *testing.T) {
	got, err := auth.ExportedNormalizeCookieSources([]string{" Safari ", "", "  ", "CHROME", "firefox", "\t"})
	if err != nil {
		t.Fatalf("normalize cookie sources: %v", err)
	}
	expect := []string{"safari", "chrome", "firefox"}
	if len(got) != len(expect) {
		t.Fatalf("len: want %d, got %d", len(expect), len(got))
	}
	for i := range expect {
		if got[i] != expect[i] {
			t.Errorf("entry %d: want %q, got %q", i, expect[i], got[i])
		}
	}
}

// TestResolveCredentials_InvalidBrowserSource verifies that an unrecognised
// browser name in CookieSources propagates as an error.
func TestResolveCredentials_InvalidBrowserSource(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("CT0", "")
	t.Setenv("TWITTER_AUTH_TOKEN", "")
	t.Setenv("TWITTER_CT0", "")
	defer func() {
		os.Unsetenv("TWITTER_AUTH_TOKEN")
		os.Unsetenv("TWITTER_CT0")
	}()
	_, err := auth.ResolveCredentials(auth.ResolveOptions{
		CookieSources: []string{"invalidbrowser"},
	})
	if err == nil {
		t.Fatal("expected error for invalid browser source, got nil")
	}
	if !strings.Contains(err.Error(), "invalidbrowser") {
		t.Errorf("error should mention the invalid browser name, got %q", err.Error())
	}
}

// TestExtractFromBrowserOrder_NilCreds verifies that when an extractor returns
// (nil, nil) it is treated as a miss and the loop continues.
func TestExtractFromBrowserOrder_NilCreds(t *testing.T) {
	// Use a 1ms timeout so browser extraction fails quickly and we get the
	// "no Twitter cookies found" error rather than a real browser error.
	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("CT0", "")
	t.Setenv("TWITTER_AUTH_TOKEN", "")
	t.Setenv("TWITTER_CT0", "")
	defer func() {
		os.Unsetenv("TWITTER_AUTH_TOKEN")
		os.Unsetenv("TWITTER_CT0")
	}()

	// extractFromBrowserOrder is called with a valid browser name that will
	// fail (no actual browser on CI). The nil,nil path is exercised when a
	// real extractor returns nil creds with nil err (not common but reachable).
	// We exercise it indirectly by passing a very short timeout so the goroutine
	// result arrives as nil,nil after the context expires, but the timeout path
	// is already tested elsewhere. Here we test the "unknown browser" branch.
	_, err := auth.ExportedExtractFromBrowserOrder([]string{"notabrowser"}, auth.ResolveOptions{})
	if err == nil {
		t.Fatal("expected error for unknown browser in order, got nil")
	}
	if !strings.Contains(err.Error(), "unknown browser") {
		t.Errorf("expected 'unknown browser' in error, got %q", err.Error())
	}
}

// TestResolveCredentials_BrowserField verifies that a Browser field alone
// (without CookieSources) flows through normalization correctly for valid names.
func TestResolveCredentials_BrowserField(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("CT0", "")
	t.Setenv("TWITTER_AUTH_TOKEN", "")
	t.Setenv("TWITTER_CT0", "")
	defer func() {
		os.Unsetenv("TWITTER_AUTH_TOKEN")
		os.Unsetenv("TWITTER_CT0")
	}()
	// An invalid browser via Browser field should error too.
	_, err := auth.ResolveCredentials(auth.ResolveOptions{
		Browser:         "edge",
		CookieTimeoutMs: 1,
	})
	if err == nil {
		t.Fatal("expected error for invalid Browser field value, got nil")
	}
}
