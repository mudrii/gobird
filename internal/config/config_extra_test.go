package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mudrii/gobird/internal/config"
)

func TestLoad_InvalidJSON5(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json5")
	if err := os.WriteFile(path, []byte(`{not valid json5 at all`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0"} {
		t.Setenv(e, "")
	}
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON5")
	}
}

func TestLoad_TrailingCommas(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trailing.json5")
	content := `{
		"authToken": "tok1",
		"ct0": "ct01",
	}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0"} {
		t.Setenv(e, "")
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("trailing commas should be valid JSON5: %v", err)
	}
	if cfg.AuthToken != "tok1" {
		t.Errorf("AuthToken: want %q, got %q", "tok1", cfg.AuthToken)
	}
}

func TestLoad_MissingFieldsGetDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.json5")
	if err := os.WriteFile(path, []byte(`{"authToken": "x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{
		"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0",
		"BIRD_TIMEOUT_MS", "BIRD_COOKIE_TIMEOUT_MS", "BIRD_QUOTE_DEPTH",
	} {
		t.Setenv(e, "")
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.QuoteDepth != 1 {
		t.Errorf("QuoteDepth: want default 1, got %d", cfg.QuoteDepth)
	}
	if cfg.TimeoutMs != 0 {
		t.Errorf("TimeoutMs: want 0, got %d", cfg.TimeoutMs)
	}
	if cfg.CookieTimeoutMs != 0 {
		t.Errorf("CookieTimeoutMs: want 0, got %d", cfg.CookieTimeoutMs)
	}
	if cfg.DefaultBrowser != "" {
		t.Errorf("DefaultBrowser: want empty, got %q", cfg.DefaultBrowser)
	}
	if cfg.ChromeProfile != "" {
		t.Errorf("ChromeProfile: want empty, got %q", cfg.ChromeProfile)
	}
	if cfg.FirefoxProfile != "" {
		t.Errorf("FirefoxProfile: want empty, got %q", cfg.FirefoxProfile)
	}
}

func TestLoad_AllFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "full.json5")
	content := `{
		"authToken": "tok",
		"ct0": "ct",
		"defaultBrowser": "chrome",
		"chromeProfile": "Profile 2",
		"chromeProfileDir": "/custom/dir",
		"firefoxProfile": "dev-edition",
		"cookieSource": ["safari", "chrome"],
		"cookieTimeoutMs": 5000,
		"timeoutMs": 10000,
		"quoteDepth": 3,
		"queryIdCachePath": "/tmp/ids.json",
		"featureOverridesPath": "/tmp/features.json"
	}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{
		"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0",
		"BIRD_TIMEOUT_MS", "BIRD_COOKIE_TIMEOUT_MS", "BIRD_QUOTE_DEPTH",
	} {
		t.Setenv(e, "")
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultBrowser != "chrome" {
		t.Errorf("DefaultBrowser: want %q, got %q", "chrome", cfg.DefaultBrowser)
	}
	if cfg.ChromeProfile != "Profile 2" {
		t.Errorf("ChromeProfile: want %q, got %q", "Profile 2", cfg.ChromeProfile)
	}
	if cfg.ChromeProfileDir != "/custom/dir" {
		t.Errorf("ChromeProfileDir: want %q, got %q", "/custom/dir", cfg.ChromeProfileDir)
	}
	if cfg.FirefoxProfile != "dev-edition" {
		t.Errorf("FirefoxProfile: want %q, got %q", "dev-edition", cfg.FirefoxProfile)
	}
	if cfg.CookieTimeoutMs != 5000 {
		t.Errorf("CookieTimeoutMs: want 5000, got %d", cfg.CookieTimeoutMs)
	}
	if cfg.TimeoutMs != 10000 {
		t.Errorf("TimeoutMs: want 10000, got %d", cfg.TimeoutMs)
	}
	if cfg.QuoteDepth != 3 {
		t.Errorf("QuoteDepth: want 3, got %d", cfg.QuoteDepth)
	}
	if cfg.QueryIDCachePath != "/tmp/ids.json" {
		t.Errorf("QueryIDCachePath: want %q, got %q", "/tmp/ids.json", cfg.QueryIDCachePath)
	}
	if cfg.FeatureOverridesPath != "/tmp/features.json" {
		t.Errorf("FeatureOverridesPath: want %q, got %q", "/tmp/features.json", cfg.FeatureOverridesPath)
	}
}

func TestLoad_EnvOverride_InvalidNumber(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json5")
	if err := os.WriteFile(path, []byte(`{"timeoutMs": 1000}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0"} {
		t.Setenv(e, "")
	}
	t.Setenv("BIRD_TIMEOUT_MS", "notanumber")
	t.Setenv("BIRD_COOKIE_TIMEOUT_MS", "")
	t.Setenv("BIRD_QUOTE_DEPTH", "")
	if _, err := config.Load(path); err == nil {
		t.Fatal("expected parse error for invalid BIRD_TIMEOUT_MS")
	}
}

func TestLoad_EnvOverride_InvalidCookieTimeoutMS(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json5")
	if err := os.WriteFile(path, []byte(`{"timeoutMs":1000}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0"} {
		t.Setenv(e, "")
	}
	t.Setenv("BIRD_COOKIE_TIMEOUT_MS", "invalid")
	t.Setenv("BIRD_TIMEOUT_MS", "")
	t.Setenv("BIRD_QUOTE_DEPTH", "")
	if _, err := config.Load(path); err == nil {
		t.Fatal("expected parse error for invalid BIRD_COOKIE_TIMEOUT_MS")
	}
}

func TestLoad_EnvOverride_InvalidQuoteDepth(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json5")
	if err := os.WriteFile(path, []byte(`{"timeoutMs":1000}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0"} {
		t.Setenv(e, "")
	}
	t.Setenv("BIRD_QUOTE_DEPTH", "not-int")
	t.Setenv("BIRD_TIMEOUT_MS", "")
	t.Setenv("BIRD_COOKIE_TIMEOUT_MS", "")
	if _, err := config.Load(path); err == nil {
		t.Fatal("expected parse error for invalid BIRD_QUOTE_DEPTH")
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json5")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{
		"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0",
		"BIRD_TIMEOUT_MS", "BIRD_COOKIE_TIMEOUT_MS", "BIRD_QUOTE_DEPTH",
	} {
		t.Setenv(e, "")
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AuthToken != "" {
		t.Errorf("AuthToken: want empty, got %q", cfg.AuthToken)
	}
	if cfg.Ct0 != "" {
		t.Errorf("Ct0: want empty, got %q", cfg.Ct0)
	}
}

func TestLoad_QuoteDepthExplicitZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "qd.json5")
	if err := os.WriteFile(path, []byte(`{"quoteDepth": 0}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{
		"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0",
		"BIRD_TIMEOUT_MS", "BIRD_COOKIE_TIMEOUT_MS", "BIRD_QUOTE_DEPTH",
	} {
		t.Setenv(e, "")
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// quoteDepth=0 in file triggers applyDefaults which sets it to 1
	if cfg.QuoteDepth != 1 {
		t.Errorf("QuoteDepth: zero should default to 1, got %d", cfg.QuoteDepth)
	}
}

func TestLoad_QuoteDepthEnvOverridesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json5")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0"} {
		t.Setenv(e, "")
	}
	t.Setenv("BIRD_TIMEOUT_MS", "")
	t.Setenv("BIRD_COOKIE_TIMEOUT_MS", "")
	t.Setenv("BIRD_QUOTE_DEPTH", "0")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.QuoteDepth != 0 {
		t.Errorf("QuoteDepth: env=0 should override default, got %d", cfg.QuoteDepth)
	}
}

func TestStringOrSlice_UnmarshalJSON_EmptyArray(t *testing.T) {
	var s config.StringOrSlice
	if err := json.Unmarshal([]byte(`[]`), &s); err != nil {
		t.Fatalf("Unmarshal empty array: %v", err)
	}
	if len(s) != 0 {
		t.Errorf("want empty slice, got %v", []string(s))
	}
}

func TestStringOrSlice_UnmarshalJSON_Boolean(t *testing.T) {
	var s config.StringOrSlice
	err := json.Unmarshal([]byte(`true`), &s)
	if err == nil {
		t.Fatal("expected error for boolean input")
	}
}

func TestStringOrSlice_UnmarshalJSON_Object(t *testing.T) {
	var s config.StringOrSlice
	err := json.Unmarshal([]byte(`{"key":"val"}`), &s)
	if err == nil {
		t.Fatal("expected error for object input")
	}
}

func TestLoad_BinaryFileContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "binary.json5")
	if err := os.WriteFile(path, []byte{0x00, 0x01, 0x02, 0xFF}, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0"} {
		t.Setenv(e, "")
	}
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for binary file content")
	}
}

func TestLoad_UnreadableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unreadable.json5")
	if err := os.WriteFile(path, []byte(`{}`), 0o000); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0"} {
		t.Setenv(e, "")
	}
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
}

func TestLoad_AUTH_TOKEN_EnvOverridesTwitterEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json5")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUTH_TOKEN", "primary")
	t.Setenv("TWITTER_AUTH_TOKEN", "secondary")
	t.Setenv("CT0", "primary_ct0")
	t.Setenv("TWITTER_CT0", "secondary_ct0")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AuthToken != "primary" {
		t.Errorf("AuthToken: AUTH_TOKEN should win over TWITTER_AUTH_TOKEN, got %q", cfg.AuthToken)
	}
	if cfg.Ct0 != "primary_ct0" {
		t.Errorf("Ct0: CT0 should win over TWITTER_CT0, got %q", cfg.Ct0)
	}
}

func TestLoad_CookieSource_EmptyString(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json5")
	if err := os.WriteFile(path, []byte(`{"cookieSource":""}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0"} {
		t.Setenv(e, "")
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CookieSource != nil {
		t.Errorf("CookieSource: empty string should be nil, got %v", cfg.CookieSource)
	}
}
