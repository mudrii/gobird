package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/mudrii/gobird/internal/auth"
	"github.com/mudrii/gobird/internal/client"
	"github.com/mudrii/gobird/internal/config"
	"github.com/mudrii/gobird/internal/output"
)

// resolveClient builds an authenticated client from global flags and config file.
func resolveClient() (*client.Client, error) {
	cfg, err := config.Load(globalFlags.configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	opts := auth.ResolveOptions{
		Browser:         globalFlags.browser,
		CookieSources:   resolveCookieSources(cfg),
		ChromeProfile:   resolveChromeProfile(cfg),
		FirefoxProfile:  firstNonEmptyString(globalFlags.firefoxProfile, cfg.FirefoxProfile),
		CookieTimeoutMs: resolveCookieTimeoutMs(cfg),
	}
	if globalFlags.authToken != "" {
		opts.FlagAuthToken = globalFlags.authToken
	} else {
		opts.FlagAuthToken = cfg.AuthToken
	}
	if globalFlags.ct0 != "" {
		opts.FlagCt0 = globalFlags.ct0
	} else {
		opts.FlagCt0 = cfg.Ct0
	}
	if opts.Browser == "" {
		opts.Browser = cfg.DefaultBrowser
	}

	creds, err := auth.ResolveCredentials(opts)
	if err != nil {
		return nil, fmt.Errorf("resolve credentials: %w", err)
	}

	var clientOpts *client.Options
	if timeoutMs := resolveTimeoutMs(cfg); timeoutMs > 0 {
		clientOpts = &client.Options{TimeoutMs: timeoutMs}
	}
	return client.New(creds.AuthToken, creds.Ct0, clientOpts), nil
}

func resolveCookieSources(cfg *config.Config) []string {
	if len(globalFlags.cookieSources) > 0 {
		return globalFlags.cookieSources
	}
	if len(cfg.CookieSource) > 0 {
		return []string(cfg.CookieSource)
	}
	if globalFlags.browser != "" {
		return []string{strings.TrimSpace(globalFlags.browser)}
	}
	if cfg.DefaultBrowser != "" {
		return []string{cfg.DefaultBrowser}
	}
	return nil
}

func resolveChromeProfile(cfg *config.Config) string {
	return firstNonEmptyString(globalFlags.chromeProfileDir, globalFlags.chromeProfile, cfg.ChromeProfileDir, cfg.ChromeProfile)
}

func resolveTimeoutMs(cfg *config.Config) int {
	if globalFlags.timeoutMs > 0 {
		return globalFlags.timeoutMs
	}
	if cfg.TimeoutMs > 0 {
		return cfg.TimeoutMs
	}
	return 0
}

func resolveCookieTimeoutMs(cfg *config.Config) int {
	if globalFlags.cookieTimeoutMs > 0 {
		return globalFlags.cookieTimeoutMs
	}
	if cfg.CookieTimeoutMs > 0 {
		return cfg.CookieTimeoutMs
	}
	return 0
}

func resolveQuoteDepth(cfg *config.Config) int {
	if globalFlags.quoteDepth >= 0 {
		return globalFlags.quoteDepth
	}
	if cfg.QuoteDepth >= 0 {
		return cfg.QuoteDepth
	}
	return 1
}

func firstNonEmptyString(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// loadMedia reads file bytes from the given path.
func loadMedia(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// detectMime returns the MIME type of the file by reading its first 512 bytes.
// Returns empty string when the type cannot be determined.
func detectMime(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck

	buf := make([]byte, 512)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return "", err
	}
	return http.DetectContentType(buf[:n]), nil
}

func currentFormatOptions() output.FormatOptions {
	return output.FormatOptions{
		Plain:   globalFlags.plain,
		NoColor: globalFlags.noColor,
		NoEmoji: globalFlags.noEmoji,
	}
}

func resolveQuoteDepthFromCommand() int {
	cfg, err := config.Load(globalFlags.configPath)
	if err != nil {
		return 1
	}
	return resolveQuoteDepth(cfg)
}

// validateOutputFlags returns an error if more than one of --json, --json-full,
// or --plain are set at the same time.
func validateOutputFlags() error {
	count := 0
	if globalFlags.jsonOutput {
		count++
	}
	if globalFlags.jsonFull {
		count++
	}
	if globalFlags.plain {
		count++
	}
	if count > 1 {
		return fmt.Errorf("invalid flags: --json, --json-full, and --plain are mutually exclusive")
	}
	return nil
}

// validateLimit returns an error when limit is negative.
func validateLimit(limit int) error {
	if limit < 0 {
		return fmt.Errorf("invalid value: --count / --limit must be >= 0, got %d", limit)
	}
	return nil
}
