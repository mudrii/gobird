package auth_test

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mudrii/gobird/internal/auth"
	"github.com/mudrii/gobird/internal/types"
)

func TestResolveFlagsWin(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "env-token")
	t.Setenv("CT0", "env-ct0")
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{
		FlagAuthToken: "flag-token",
		FlagCt0:       "flag-ct0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AuthToken != "flag-token" {
		t.Errorf("want flag-token, got %q", creds.AuthToken)
	}
	if creds.Ct0 != "flag-ct0" {
		t.Errorf("want flag-ct0, got %q", creds.Ct0)
	}
}

func TestResolveEnvVarsWin(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "env-token")
	t.Setenv("CT0", "env-ct0")
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AuthToken != "env-token" {
		t.Errorf("want env-token, got %q", creds.AuthToken)
	}
}

func TestResolveTwitterEnvFallback(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("CT0", "")
	t.Setenv("TWITTER_AUTH_TOKEN", "tw-token")
	t.Setenv("TWITTER_CT0", "tw-ct0")
	defer func() {
		os.Unsetenv("TWITTER_AUTH_TOKEN")
		os.Unsetenv("TWITTER_CT0")
	}()
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AuthToken != "tw-token" {
		t.Errorf("want tw-token, got %q", creds.AuthToken)
	}
}

func TestResolveFlagsNeedBoth(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "env-token")
	t.Setenv("CT0", "env-ct0")
	// Only one flag set — should fall through to env.
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{FlagAuthToken: "flag-only"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should use env vars since flag was incomplete.
	if creds.AuthToken != "env-token" {
		t.Errorf("want env-token, got %q", creds.AuthToken)
	}
}

func TestCookieHeader(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "tok")
	t.Setenv("CT0", "c0")
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "auth_token=tok; ct0=c0"
	if creds.CookieHeader != want {
		t.Errorf("CookieHeader: want %q, got %q", want, creds.CookieHeader)
	}
}

// TestResolveCredentials_FlagAuthTokenAndCt0 verifies that when both flags are
// set they are returned directly, bypassing env and browser extraction.
func TestResolveCredentials_FlagAuthTokenAndCt0(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("CT0", "")
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{
		FlagAuthToken: "myflagtoken",
		FlagCt0:       "myflagct0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AuthToken != "myflagtoken" {
		t.Errorf("AuthToken: want %q, got %q", "myflagtoken", creds.AuthToken)
	}
	if creds.Ct0 != "myflagct0" {
		t.Errorf("Ct0: want %q, got %q", "myflagct0", creds.Ct0)
	}
	if creds.CookieHeader != "auth_token=myflagtoken; ct0=myflagct0" {
		t.Errorf("CookieHeader: got %q", creds.CookieHeader)
	}
}

// TestResolveCredentials_FlagOnlyOneSet verifies that when only one flag is set
// the resolver falls through to env vars.
func TestResolveCredentials_FlagOnlyOneSet(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "envtok")
	t.Setenv("CT0", "envct0")
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{
		FlagAuthToken: "partial",
		// FlagCt0 deliberately omitted
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AuthToken != "envtok" {
		t.Errorf("expected env fallback, got %q", creds.AuthToken)
	}
}

// TestResolveCredentials_EnvVars verifies AUTH_TOKEN and CT0 env vars are used.
func TestResolveCredentials_EnvVars(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "myauthtoken")
	t.Setenv("CT0", "myct0val")
	t.Setenv("TWITTER_AUTH_TOKEN", "")
	t.Setenv("TWITTER_CT0", "")
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AuthToken != "myauthtoken" {
		t.Errorf("AuthToken: want %q, got %q", "myauthtoken", creds.AuthToken)
	}
	if creds.Ct0 != "myct0val" {
		t.Errorf("Ct0: want %q, got %q", "myct0val", creds.Ct0)
	}
}

// TestResolveCredentials_TwitterEnvVars verifies TWITTER_AUTH_TOKEN and
// TWITTER_CT0 are used when the primary env vars are absent.
func TestResolveCredentials_TwitterEnvVars(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("CT0", "")
	t.Setenv("TWITTER_AUTH_TOKEN", "twtoken")
	t.Setenv("TWITTER_CT0", "twct0")
	defer func() {
		os.Unsetenv("TWITTER_AUTH_TOKEN")
		os.Unsetenv("TWITTER_CT0")
	}()
	creds, err := auth.ResolveCredentials(auth.ResolveOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AuthToken != "twtoken" {
		t.Errorf("AuthToken: want %q, got %q", "twtoken", creds.AuthToken)
	}
	if creds.Ct0 != "twct0" {
		t.Errorf("Ct0: want %q, got %q", "twct0", creds.Ct0)
	}
}

// TestResolveCredentials_NoCredentials verifies that when no flags, no env vars,
// and no browser cookies are available, an error is returned.
func TestResolveCredentials_NoCredentials(t *testing.T) {
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
	cases := []struct {
		token string
		ct0   string
		want  string
	}{
		{"abc", "xyz", "auth_token=abc; ct0=xyz"},
		{"tok123", "ct0val", "auth_token=tok123; ct0=ct0val"},
	}
	for _, tc := range cases {
		t.Run(tc.token, func(t *testing.T) {
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
			name: "primary wins over twitter env",
			authToken: "primary", twitterAT: "twitter",
			ct0: "pct0", twitterCt: "tct0",
			wantToken: "primary", wantCt0: "pct0",
		},
		{
			name: "falls back to twitter env when primary empty",
			authToken: "", twitterAT: "twitter",
			ct0: "", twitterCt: "tct0",
			wantToken: "twitter", wantCt0: "tct0",
		},
		{
			name: "both empty leads to browser attempt and error",
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
	got, err := auth.ExportedExtractWithTimeout(500, func() (*types.TwitterCookies, error) {
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
	got, err := auth.ExportedExtractWithTimeout(0, func() (*types.TwitterCookies, error) {
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

	_, err := auth.ExportedExtractWithTimeout(20, func() (*types.TwitterCookies, error) {
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		select {
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
