package auth

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/mudrii/gobird/internal/types"
)

// extractFirefox reads cookies from Firefox's plain SQLite cookie store.
func extractFirefox(profileHint string) (*types.TwitterCookies, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("Firefox: home directory: %w", err)
	}

	profileDir := filepath.Join(home, "Library", "Application Support", "Firefox", "Profiles")
	entries, err := os.ReadDir(profileDir)
	if err != nil {
		return nil, fmt.Errorf("Firefox: profile directory not found: %w", err)
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
		return nil, fmt.Errorf("Firefox: no cookies.sqlite found")
	}

	var cookies []domainCookie
	for _, dbPath := range dbPaths {
		c, err := readFirefoxCookies(dbPath)
		if err != nil {
			continue
		}
		cookies = append(cookies, c...)
	}

	authToken, ct0 := preferredDomainCookies(cookies)
	if authToken == "" || ct0 == "" {
		return nil, fmt.Errorf("Firefox: auth_token or ct0 not found")
	}
	return &types.TwitterCookies{
		AuthToken:    authToken,
		Ct0:          ct0,
		CookieHeader: buildCookieHeader(authToken, ct0),
	}, nil
}

func readFirefoxCookies(dbPath string) ([]domainCookie, error) {
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro&immutable=1")
	if err != nil {
		return nil, fmt.Errorf("Firefox: open cookie database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(
		`SELECT host, name, value FROM moz_cookies WHERE name IN ('auth_token','ct0') AND (host LIKE '%x.com' OR host LIKE '%twitter.com')`,
	)
	if err != nil {
		return nil, fmt.Errorf("Firefox: query cookies: %w", err)
	}
	defer rows.Close()

	var result []domainCookie
	for rows.Next() {
		var c domainCookie
		if err := rows.Scan(&c.domain, &c.name, &c.value); err != nil {
			continue
		}
		result = append(result, c)
	}
	return result, rows.Err()
}
