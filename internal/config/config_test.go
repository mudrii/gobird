package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mudrii/gobird/internal/config"
)

func TestLoadEmptyWhenNoFile(t *testing.T) {
	// Unset all env vars that could interfere.
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0", "BIRD_CONFIG"} {
		t.Setenv(e, "")
	}
	cfg, err := config.Load("/nonexistent/path/bird.json5")
	if err == nil {
		t.Fatalf("expected error for nonexistent path, got nil with cfg=%+v", cfg)
	}
}

func TestLoadJSON5File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json5")
	content := `{
		// bird config
		"authToken": "mytoken",
		"ct0": "myct0",
	}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0"} {
		t.Setenv(e, "")
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AuthToken != "mytoken" {
		t.Errorf("AuthToken: want %q, got %q", "mytoken", cfg.AuthToken)
	}
	if cfg.Ct0 != "myct0" {
		t.Errorf("Ct0: want %q, got %q", "myct0", cfg.Ct0)
	}
}

func TestEnvVarPrecedence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json5")
	content := `{"authToken":"file-token","ct0":"file-ct0"}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUTH_TOKEN", "env-token")
	t.Setenv("CT0", "env-ct0")
	defer func() {
		os.Unsetenv("AUTH_TOKEN")
		os.Unsetenv("CT0")
	}()

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AuthToken != "env-token" {
		t.Errorf("AuthToken: want env-token, got %q", cfg.AuthToken)
	}
	if cfg.Ct0 != "env-ct0" {
		t.Errorf("Ct0: want env-ct0, got %q", cfg.Ct0)
	}
}

func TestTwitterEnvVarFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json5")
	content := `{}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("TWITTER_AUTH_TOKEN", "tw-token")
	t.Setenv("CT0", "")
	t.Setenv("TWITTER_CT0", "tw-ct0")
	defer func() {
		os.Unsetenv("TWITTER_AUTH_TOKEN")
		os.Unsetenv("TWITTER_CT0")
	}()

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AuthToken != "tw-token" {
		t.Errorf("AuthToken: want tw-token, got %q", cfg.AuthToken)
	}
	if cfg.Ct0 != "tw-ct0" {
		t.Errorf("Ct0: want tw-ct0, got %q", cfg.Ct0)
	}
}

// TestLoad_ExplicitPath_NotFound verifies that a non-existent explicit path
// returns an error rather than silently returning an empty config.
func TestLoad_ExplicitPath_NotFound(t *testing.T) {
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0", "BIRD_CONFIG"} {
		t.Setenv(e, "")
	}
	_, err := config.Load("/tmp/definitely-does-not-exist-gobird-test.json5")
	if err == nil {
		t.Fatal("expected error for missing explicit path, got nil")
	}
}

// TestLoad_EnvVarPath verifies that BIRD_CONFIG env var is honoured.
func TestLoad_EnvVarPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bird.json5")
	if err := os.WriteFile(path, []byte(`{"authToken":"envpath-tok","ct0":"envpath-ct0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BIRD_CONFIG", path)
	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("CT0", "")
	t.Setenv("TWITTER_AUTH_TOKEN", "")
	t.Setenv("TWITTER_CT0", "")
	defer func() {
		os.Unsetenv("BIRD_CONFIG")
		os.Unsetenv("TWITTER_AUTH_TOKEN")
		os.Unsetenv("TWITTER_CT0")
	}()
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AuthToken != "envpath-tok" {
		t.Errorf("AuthToken: want %q, got %q", "envpath-tok", cfg.AuthToken)
	}
}

// TestLoad_DefaultsApplied verifies that an empty config file still gets
// default values applied (e.g. QuoteDepth defaults to 1).
func TestLoad_DefaultsApplied(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json5")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0",
		"BIRD_TIMEOUT_MS", "BIRD_COOKIE_TIMEOUT_MS", "BIRD_QUOTE_DEPTH"} {
		t.Setenv(e, "")
	}
	defer func() {
		os.Unsetenv("TWITTER_AUTH_TOKEN")
		os.Unsetenv("TWITTER_CT0")
		os.Unsetenv("BIRD_TIMEOUT_MS")
		os.Unsetenv("BIRD_COOKIE_TIMEOUT_MS")
		os.Unsetenv("BIRD_QUOTE_DEPTH")
	}()
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.QuoteDepth != 1 {
		t.Errorf("QuoteDepth default: want 1, got %d", cfg.QuoteDepth)
	}
}

// TestLoad_StringOrSlice_String verifies that a single JSON string is parsed
// into a one-element []string.
func TestLoad_StringOrSlice_String(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json5")
	if err := os.WriteFile(path, []byte(`{"cookieSource":"safari"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0"} {
		t.Setenv(e, "")
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.CookieSource) != 1 || cfg.CookieSource[0] != "safari" {
		t.Errorf("CookieSource: want [safari], got %v", cfg.CookieSource)
	}
}

// TestLoad_StringOrSlice_Array verifies that a JSON array is parsed into a
// []string with all elements preserved.
func TestLoad_StringOrSlice_Array(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json5")
	if err := os.WriteFile(path, []byte(`{"cookieSource":["chrome","firefox"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0"} {
		t.Setenv(e, "")
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.CookieSource) != 2 || cfg.CookieSource[0] != "chrome" || cfg.CookieSource[1] != "firefox" {
		t.Errorf("CookieSource: want [chrome firefox], got %v", cfg.CookieSource)
	}
}

// TestLoad_EnvOverrides verifies that BIRD_TIMEOUT_MS, BIRD_COOKIE_TIMEOUT_MS,
// and BIRD_QUOTE_DEPTH override file-loaded values.
func TestLoad_EnvOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json5")
	content := `{"timeoutMs":1000,"cookieTimeoutMs":2000,"quoteDepth":3}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("CT0", "")
	t.Setenv("TWITTER_AUTH_TOKEN", "")
	t.Setenv("TWITTER_CT0", "")
	t.Setenv("BIRD_TIMEOUT_MS", "5000")
	t.Setenv("BIRD_COOKIE_TIMEOUT_MS", "6000")
	t.Setenv("BIRD_QUOTE_DEPTH", "7")
	defer func() {
		os.Unsetenv("TWITTER_AUTH_TOKEN")
		os.Unsetenv("TWITTER_CT0")
		os.Unsetenv("BIRD_TIMEOUT_MS")
		os.Unsetenv("BIRD_COOKIE_TIMEOUT_MS")
		os.Unsetenv("BIRD_QUOTE_DEPTH")
	}()
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TimeoutMs != 5000 {
		t.Errorf("TimeoutMs: want 5000, got %d", cfg.TimeoutMs)
	}
	if cfg.CookieTimeoutMs != 6000 {
		t.Errorf("CookieTimeoutMs: want 6000, got %d", cfg.CookieTimeoutMs)
	}
	if cfg.QuoteDepth != 7 {
		t.Errorf("QuoteDepth: want 7, got %d", cfg.QuoteDepth)
	}
}

// TestLoad_EmptyWhenNoFileFound verifies that when no explicit path, no
// BIRD_CONFIG env var, and no default config files exist, Load returns an
// empty Config with no error.
func TestLoad_EmptyWhenNoFileFound(t *testing.T) {
	for _, e := range []string{
		"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0", "BIRD_CONFIG",
		"BIRD_TIMEOUT_MS", "BIRD_COOKIE_TIMEOUT_MS", "BIRD_QUOTE_DEPTH",
	} {
		t.Setenv(e, "")
	}
	defer func() {
		os.Unsetenv("TWITTER_AUTH_TOKEN")
		os.Unsetenv("TWITTER_CT0")
		os.Unsetenv("BIRD_TIMEOUT_MS")
		os.Unsetenv("BIRD_COOKIE_TIMEOUT_MS")
		os.Unsetenv("BIRD_QUOTE_DEPTH")
	}()

	// Pass empty string so Load uses default paths. In a clean temp dir there
	// are no default config files, so it should return a defaulted Config.
	// We cannot fully control the default paths (they depend on $HOME and cwd),
	// so we simply verify no error is returned and that QuoteDepth got its default.
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load with no file: unexpected error: %v", err)
	}
	// QuoteDepth default should be applied even if no file was loaded.
	if cfg.QuoteDepth != 1 {
		t.Errorf("QuoteDepth: want default 1, got %d", cfg.QuoteDepth)
	}
}
