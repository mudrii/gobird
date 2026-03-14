package auth

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // register SQLite driver for database/sql

	"github.com/mudrii/gobird/internal/types"
)

// extractSafari reads cookies from Safari's binary Cookies.binarycookies file
// via the SQLite-based WebKit cookie store on macOS.
func extractSafari() (result *types.TwitterCookies, err error) {
	return extractSafariWithContext(context.Background())
}

func extractSafariWithContext(ctx context.Context) (result *types.TwitterCookies, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("safari: home directory: %w", err)
	}
	// Safari on macOS stores cookies in a SQLite DB under Library/Cookies/Cookies.db.
	dbPath := filepath.Join(home, "Library", "Containers", "com.apple.Safari", "Data",
		"Library", "Cookies", "Cookies.db")
	if _, err := os.Stat(dbPath); err != nil {
		// Try alternate path for non-sandboxed Safari.
		dbPath = filepath.Join(home, "Library", "Cookies", "Cookies.db")
		if _, err2 := os.Stat(dbPath); err2 != nil {
			return nil, fmt.Errorf("safari: cookie database not found")
		}
	}
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro&immutable=1")
	if err != nil {
		return nil, fmt.Errorf("safari: open cookie database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("safari: close cookie database: %w", closeErr)
		}
	}()

	rows, err := db.QueryContext(ctx,
		`SELECT domain, name, value FROM cookies WHERE name IN ('auth_token','ct0') AND (domain LIKE '%x.com' OR domain LIKE '%twitter.com')`,
	)
	if err != nil {
		return nil, fmt.Errorf("safari: query cookies: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("safari: close cookies query: %w", closeErr)
		}
	}()

	var cookies []domainCookie
	for rows.Next() {
		var c domainCookie
		if err := rows.Scan(&c.domain, &c.name, &c.value); err != nil {
			continue
		}
		cookies = append(cookies, c)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("safari: iterate cookies: %w", rows.Err())
	}

	authToken, ct0 := preferredDomainCookies(cookies)
	if authToken == "" || ct0 == "" {
		return nil, fmt.Errorf("safari: auth_token or ct0 not found")
	}
	result = &types.TwitterCookies{
		AuthToken:    authToken,
		Ct0:          ct0,
		CookieHeader: buildCookieHeader(authToken, ct0),
	}
	return result, nil
}
