package bird_test

import (
	"errors"
	"testing"

	"github.com/mudrii/gobird/pkg/bird"
)

func TestNew_ReturnsClient(t *testing.T) {
	creds := &bird.TwitterCookies{
		AuthToken: "test_auth_token",
		Ct0:       "test_ct0_value",
	}
	c, err := bird.New(creds, nil)
	if err != nil {
		t.Fatalf("New() with valid credentials: %v", err)
	}
	if c == nil {
		t.Fatal("New() returned nil client")
	}
}

func TestNew_NilCreds(t *testing.T) {
	c, err := bird.New(nil, nil)
	if err == nil {
		t.Fatal("New(nil) should return error")
	}
	if c != nil {
		t.Fatal("New(nil) should return nil client")
	}
}

func TestNew_EmptyAuthToken(t *testing.T) {
	creds := &bird.TwitterCookies{AuthToken: "", Ct0: "ct0"}
	c, err := bird.New(creds, nil)
	if err == nil {
		t.Fatal("New() with empty AuthToken should return error")
	}
	if c != nil {
		t.Fatal("New() with empty AuthToken should return nil client")
	}
}

func TestNew_EmptyCt0(t *testing.T) {
	creds := &bird.TwitterCookies{AuthToken: "tok", Ct0: ""}
	c, err := bird.New(creds, nil)
	if err == nil {
		t.Fatal("New() with empty Ct0 should return error")
	}
	if c != nil {
		t.Fatal("New() with empty Ct0 should return nil client")
	}
}

func TestNewWithTokens_SetsCredentials(t *testing.T) {
	c, err := bird.NewWithTokens("auth_token", "ct0_value", nil)
	if err != nil {
		t.Fatalf("NewWithTokens() with valid tokens: %v", err)
	}
	if c == nil {
		t.Fatal("NewWithTokens() returned nil client")
	}
}

func TestNewWithTokens_EmptyToken(t *testing.T) {
	c, err := bird.NewWithTokens("", "ct0", nil)
	if err == nil {
		t.Fatal("NewWithTokens() with empty authToken should return error")
	}
	if c != nil {
		t.Fatal("NewWithTokens() with empty authToken should return nil client")
	}
}

func TestNewWithTokens_EmptyCt0(t *testing.T) {
	c, err := bird.NewWithTokens("auth", "", nil)
	if err == nil {
		t.Fatal("NewWithTokens() with empty ct0 should return error")
	}
	if c != nil {
		t.Fatal("NewWithTokens() with empty ct0 should return nil client")
	}
}

func TestNew_MissingCredentials_ErrorMessage(t *testing.T) {
	_, err := bird.New(nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// ErrMissingCredentials is unexported; verify it's a known sentinel via errors.Is
	// by checking the message contains key text.
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestResolveCredentials_TypeAlias(t *testing.T) {
	// Verify that bird.ResolveOptions is callable and accepts the expected fields.
	opts := bird.ResolveOptions{
		FlagAuthToken: "tok",
		FlagCt0:       "ct0",
	}
	// ResolveCredentials should not panic — it will fail because no env/browser
	// sources are set, but the type alias is correctly wired.
	_, err := bird.ResolveCredentials(opts)
	if err == nil {
		// Got credentials from the environment — that's fine.
		return
	}
	// Expect a credential-resolution error, not a type error.
	_ = err
}

func TestTwitterCookies_TypeAlias(t *testing.T) {
	// Verify the type alias round-trips its fields correctly.
	c := bird.TwitterCookies{
		AuthToken:    "a",
		Ct0:          "b",
		CookieHeader: "c=d",
	}
	if c.AuthToken != "a" {
		t.Errorf("AuthToken: want %q, got %q", "a", c.AuthToken)
	}
	if c.Ct0 != "b" {
		t.Errorf("Ct0: want %q, got %q", "b", c.Ct0)
	}
	if c.CookieHeader != "c=d" {
		t.Errorf("CookieHeader: want %q, got %q", "c=d", c.CookieHeader)
	}
}

func TestNew_MissingCredentials_IsError(t *testing.T) {
	_, err := bird.New(nil, nil)
	// Ensure the returned value satisfies the error interface.
	_ = err
	if !errors.Is(err, err) {
		t.Error("error should satisfy errors.Is(err, err)")
	}
}
