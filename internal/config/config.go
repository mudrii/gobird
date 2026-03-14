// Package config handles JSON5 configuration file loading and env var resolution.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/tailscale/hujson"
)

// StringOrSlice decodes either a JSON string or string array into a slice.
type StringOrSlice []string

// UnmarshalJSON accepts `"value"` or `["value"]`.
func (s *StringOrSlice) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		if single == "" {
			*s = nil
		} else {
			*s = []string{single}
		}
		return nil
	}

	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*s = many
	return nil
}

// Config holds all configurable gobird settings.
type Config struct {
	// AuthToken is the Twitter auth_token cookie value.
	AuthToken string `json:"authToken"`
	// Ct0 is the Twitter ct0 cookie value.
	Ct0 string `json:"ct0"`
	// DefaultBrowser selects which browser to extract cookies from ("safari", "chrome", "firefox").
	DefaultBrowser string `json:"defaultBrowser"`
	// ChromeProfile selects the Chrome profile name for cookie extraction.
	ChromeProfile string `json:"chromeProfile"`
	// ChromeProfileDir selects the Chrome/Chromium profile directory or cookie DB path.
	ChromeProfileDir string `json:"chromeProfileDir"`
	// FirefoxProfile selects the Firefox profile name for cookie extraction.
	FirefoxProfile string `json:"firefoxProfile"`
	// CookieSource controls browser cookie source order.
	CookieSource StringOrSlice `json:"cookieSource"`
	// CookieTimeoutMs controls browser cookie extraction timeout.
	CookieTimeoutMs int `json:"cookieTimeoutMs"`
	// TimeoutMs controls HTTP request timeout.
	TimeoutMs int `json:"timeoutMs"`
	// QuoteDepth controls quoted tweet expansion depth.
	QuoteDepth int `json:"quoteDepth"`
	// QueryIDCachePath overrides the default query ID cache file location.
	QueryIDCachePath string `json:"queryIdCachePath"`
	// FeatureOverridesPath overrides the default features JSON path.
	FeatureOverridesPath string `json:"featureOverridesPath"`
}

// Load reads and parses the gobird config files, applying env var overrides.
// Order: explicit path or $BIRD_CONFIG; otherwise global ~/.config/gobird/config.json5
// followed by local ./.gobirdrc.json5, with local overriding global.
func Load(explicitPath string) (*Config, error) {
	cfg := &Config{}
	if path := explicitOrEnvPath(explicitPath); path != "" {
		if err := loadFile(path, cfg); err != nil {
			return nil, err
		}
	} else {
		paths, err := defaultConfigPaths()
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			if path == "" {
				continue
			}
			if _, err := os.Stat(path); err == nil {
				if err := loadFile(path, cfg); err != nil {
					return nil, err
				}
			}
		}
	}
	applyDefaults(cfg)
	if err := applyEnv(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func explicitOrEnvPath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if v := os.Getenv("BIRD_CONFIG"); v != "" {
		return v
	}
	return ""
}

func defaultConfigPaths() ([]string, error) {
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return nil, fmt.Errorf("resolve home directory: %w", homeErr)
	}
	cwd, cwdErr := os.Getwd()
	if cwdErr != nil {
		return []string{
			filepath.Join(home, ".config", "gobird", "config.json5"),
		}, nil
	}
	return []string{
		filepath.Join(home, ".config", "gobird", "config.json5"),
		filepath.Join(cwd, ".gobirdrc.json5"),
	}, nil
}

func loadFile(path string, cfg *Config) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}
	// Normalize JSON5 → standard JSON.
	std, err := hujson.Standardize(b)
	if err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	if err := json.Unmarshal(std, cfg); err != nil {
		return fmt.Errorf("unmarshal config %s: %w", path, err)
	}
	return nil
}

// applyDefaults sets zero-value fields to their documented defaults.
func applyDefaults(cfg *Config) {
	if cfg.QuoteDepth == 0 {
		cfg.QuoteDepth = 1
	}
}

// applyEnv overlays environment variables onto the loaded config.
// Env vars take precedence over file values.
func applyEnv(cfg *Config) error {
	// Credential resolution order: AUTH_TOKEN > TWITTER_AUTH_TOKEN.
	if v := os.Getenv("AUTH_TOKEN"); v != "" {
		cfg.AuthToken = v
	} else if v := os.Getenv("TWITTER_AUTH_TOKEN"); v != "" {
		cfg.AuthToken = v
	}
	// CT0 > TWITTER_CT0.
	if v := os.Getenv("CT0"); v != "" {
		cfg.Ct0 = v
	} else if v := os.Getenv("TWITTER_CT0"); v != "" {
		cfg.Ct0 = v
	}
	if v := os.Getenv("BIRD_TIMEOUT_MS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid value for BIRD_TIMEOUT_MS: %q; expected integer milliseconds", v)
		}
		cfg.TimeoutMs = n
	}
	if v := os.Getenv("BIRD_COOKIE_TIMEOUT_MS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid value for BIRD_COOKIE_TIMEOUT_MS: %q; expected integer milliseconds", v)
		}
		cfg.CookieTimeoutMs = n
	}
	if v := os.Getenv("BIRD_QUOTE_DEPTH"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid value for BIRD_QUOTE_DEPTH: %q; expected integer", v)
		}
		cfg.QuoteDepth = n
	}
	return nil
}
