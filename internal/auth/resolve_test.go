package auth_test

import (
	"os"
	"testing"

	"github.com/mudrii/gobird/internal/auth"
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
