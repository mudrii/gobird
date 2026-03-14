package auth

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // register SQLite driver for database/sql

	"github.com/mudrii/gobird/internal/types"
)

// extractFirefox reads cookies from Firefox's plain SQLite cookie store.
func extractFirefox(profileHint string) (result *types.TwitterCookies, err error) {
	return extractFirefoxWithContext(context.Background(), profileHint)
}

func extractFirefoxWithContext(ctx context.Context, profileHint string) (result *types.TwitterCookies, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("firefox: home directory: %w", err)
	}

	profileDir := filepath.Join(home, "Library", "Application Support", "Firefox", "Profiles")
	entries, err := os.ReadDir(profileDir)
	if err != nil {
		return nil, fmt.Errorf("firefox: profile directory not found: %w", err)
	}

	var dbPaths []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if profileHint != "" {
			name := strings.TrimSpace(profileHint)
			if name != "" && e.Name() != name && !strings.Contains(e.Name(), name) {
				continue
			}
		}
		p := filepath.Join(profileDir, e.Name(), "cookies.sqlite")
		if _, err := os.Stat(p); err == nil {
			dbPaths = append(dbPaths, p)
		}
	}
	if len(dbPaths) == 0 {
		return nil, fmt.Errorf("firefox: no cookies.sqlite found")
	}

	var cookies []domainCookie
	var lastDBErr error
	for _, dbPath := range dbPaths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		c, err := readFirefoxCookiesWithContext(ctx, dbPath)
		if err != nil {
			lastDBErr = err
			continue
		}
		cookies = append(cookies, c...)
	}

	authToken, ct0 := preferredDomainCookies(cookies)
	if authToken == "" || ct0 == "" {
		if lastDBErr != nil {
			return nil, fmt.Errorf("firefox: auth_token or ct0 not found: %w", lastDBErr)
		}
		return nil, fmt.Errorf("firefox: auth_token or ct0 not found")
	}
	result = &types.TwitterCookies{
		AuthToken:    authToken,
		Ct0:          ct0,
		CookieHeader: buildCookieHeader(authToken, ct0),
	}
	return result, nil
}

func readFirefoxCookies(dbPath string) (result []domainCookie, err error) {
	return readFirefoxCookiesWithContext(context.Background(), dbPath)
}

func readFirefoxCookiesWithContext(ctx context.Context, dbPath string) (result []domainCookie, err error) {
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro&immutable=1")
	if err != nil {
		return nil, fmt.Errorf("firefox: open cookie database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("firefox: close cookie database: %w", closeErr)
		}
	}()

	rows, err := db.QueryContext(ctx,
		`SELECT host, name, value FROM moz_cookies WHERE name IN ('auth_token','ct0') AND (host LIKE '%x.com' OR host LIKE '%twitter.com')`,
	)
	if err != nil {
		return nil, fmt.Errorf("firefox: query cookies: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("firefox: close cookies query: %w", closeErr)
		}
	}()

	for rows.Next() {
		var c domainCookie
		if err := rows.Scan(&c.domain, &c.name, &c.value); err != nil {
			continue
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("firefox: iterate cookies: %w", err)
	}
	return result, nil
}
