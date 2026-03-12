package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/mudrii/gobird/internal/auth"
	"github.com/mudrii/gobird/internal/client"
	"github.com/mudrii/gobird/internal/config"
)

// resolveClient builds an authenticated client from global flags and config file.
func resolveClient() (*client.Client, error) {
	cfg, err := config.Load(globalFlags.configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	opts := auth.ResolveOptions{
		Browser: globalFlags.browser,
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

	return client.New(creds.AuthToken, creds.Ct0, nil), nil
}

// resolveCredentialsFromOptions resolves credentials using current global flags and config.
func resolveCredentialsFromOptions() (authToken, ct0 string, err error) {
	cfg, err := config.Load(globalFlags.configPath)
	if err != nil {
		return "", "", fmt.Errorf("load config: %w", err)
	}

	opts := auth.ResolveOptions{
		Browser: globalFlags.browser,
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
		return "", "", fmt.Errorf("resolve credentials: %w", err)
	}
	return creds.AuthToken, creds.Ct0, nil
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
	defer f.Close()

	buf := make([]byte, 512)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return "", err
	}
	return http.DetectContentType(buf[:n]), nil
}
