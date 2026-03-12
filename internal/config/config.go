// Package config handles JSON5 configuration file loading and env var resolution.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/tailscale/hujson"
)

// Config holds all configurable bird settings.
type Config struct {
	// AuthToken is the Twitter auth_token cookie value.
	AuthToken string `json:"authToken"`
	// Ct0 is the Twitter ct0 cookie value.
	Ct0 string `json:"ct0"`
	// DefaultBrowser selects which browser to extract cookies from ("safari", "chrome", "firefox").
	DefaultBrowser string `json:"defaultBrowser"`
	// QueryIDCachePath overrides the default query ID cache file location.
	QueryIDCachePath string `json:"queryIdCachePath"`
	// FeatureOverridesPath overrides the default features JSON path.
	FeatureOverridesPath string `json:"featureOverridesPath"`
}

// Load reads and parses the bird config file, applying env var overrides.
// Searches in order: explicit path → $BIRD_CONFIG → ~/.config/bird/config.json5 → ~/.bird.json5.
// Returns an empty Config (not an error) when no file is found.
func Load(explicitPath string) (*Config, error) {
	path := resolvePath(explicitPath)
	cfg := &Config{}
	if path != "" {
		if err := loadFile(path, cfg); err != nil {
			return nil, err
		}
	}
	applyEnv(cfg)
	return cfg, nil
}

func resolvePath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if v := os.Getenv("BIRD_CONFIG"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	candidates := []string{
		filepath.Join(home, ".config", "bird", "config.json5"),
		filepath.Join(home, ".bird.json5"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func loadFile(path string, cfg *Config) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// Normalize JSON5 → standard JSON.
	std, err := hujson.Standardize(b)
	if err != nil {
		return err
	}
	return json.Unmarshal(std, cfg)
}

// applyEnv overlays environment variables onto the loaded config.
// Env vars take precedence over file values.
func applyEnv(cfg *Config) {
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
}
