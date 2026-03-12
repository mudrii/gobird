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
