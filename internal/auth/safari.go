package auth

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // register SQLite driver for database/sql

	"github.com/mudrii/gobird/internal/types"
)

const safariBinaryCookiesMagic = "cook"

// extractSafari reads cookies from Safari's WebKit cookie store on macOS.
// Modern Safari uses Cookies.binarycookies; older installations may still use Cookies.db.
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

	storePath, err := findSafariCookieStore(home)
	if err != nil {
		return nil, err
	}

	var cookies []domainCookie
	switch filepath.Ext(storePath) {
	case ".binarycookies":
		cookies, err = readSafariBinaryCookies(ctx, storePath)
	case ".db":
		cookies, err = readSafariSQLiteCookies(ctx, storePath)
	default:
		err = fmt.Errorf("safari: unsupported cookie store: %s", filepath.Base(storePath))
	}
	if err != nil {
		return nil, err
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

func findSafariCookieStore(home string) (string, error) {
	candidates := []string{
		filepath.Join(home, "Library", "Containers", "com.apple.Safari", "Data", "Library", "Cookies", "Cookies.binarycookies"),
		filepath.Join(home, "Library", "Cookies", "Cookies.binarycookies"),
		filepath.Join(home, "Library", "Containers", "com.apple.Safari", "Data", "Library", "Cookies", "Cookies.db"),
		filepath.Join(home, "Library", "Cookies", "Cookies.db"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("safari: cookie store not found")
}

func readSafariSQLiteCookies(ctx context.Context, dbPath string) (cookies []domainCookie, err error) {
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

	for rows.Next() {
		var c domainCookie
		if err := rows.Scan(&c.domain, &c.name, &c.value); err != nil {
			continue
		}
		c.domain = normalizeTwitterCookieDomain(c.domain)
		cookies = append(cookies, c)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("safari: iterate cookies: %w", rows.Err())
	}
	return cookies, nil
}

func readSafariBinaryCookies(ctx context.Context, cookiePath string) ([]domainCookie, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	//nolint:gosec // cookiePath comes from a fixed Safari store candidate list, not user input.
	data, err := os.ReadFile(cookiePath)
	if err != nil {
		return nil, fmt.Errorf("safari: read binary cookie store: %w", err)
	}
	cookies, err := parseSafariBinaryCookies(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("safari: parse binary cookie store: %w", err)
	}
	return cookies, nil
}

func parseSafariBinaryCookies(ctx context.Context, data []byte) ([]domainCookie, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("truncated header")
	}
	if string(data[:4]) != safariBinaryCookiesMagic {
		return nil, fmt.Errorf("invalid header")
	}

	pageCount := int(binary.BigEndian.Uint32(data[4:8]))
	if pageCount < 0 {
		return nil, fmt.Errorf("invalid page count")
	}
	headerSize := 8 + pageCount*4
	if len(data) < headerSize {
		return nil, fmt.Errorf("truncated page table")
	}

	var cookies []domainCookie
	offset := headerSize
	for i := 0; i < pageCount; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pageSize := int(binary.BigEndian.Uint32(data[8+i*4 : 12+i*4]))
		if pageSize <= 0 {
			return nil, fmt.Errorf("invalid page size")
		}
		if offset+pageSize > len(data) {
			return nil, fmt.Errorf("truncated page data")
		}
		pageCookies, err := parseSafariBinaryCookiePage(data[offset : offset+pageSize])
		if err != nil {
			return nil, err
		}
		cookies = append(cookies, pageCookies...)
		offset += pageSize
	}
	return cookies, nil
}

func parseSafariBinaryCookiePage(page []byte) ([]domainCookie, error) {
	if len(page) < 12 {
		return nil, fmt.Errorf("truncated cookie page")
	}

	cookieCount := int(binary.LittleEndian.Uint32(page[4:8]))
	offsetTableEnd := 8 + cookieCount*4
	if len(page) < offsetTableEnd {
		return nil, fmt.Errorf("truncated cookie offset table")
	}

	cookies := make([]domainCookie, 0, cookieCount)
	for i := 0; i < cookieCount; i++ {
		cookieOffset := int(binary.LittleEndian.Uint32(page[8+i*4 : 12+i*4]))
		if cookieOffset <= 0 || cookieOffset+4 > len(page) {
			continue
		}
		cookie, ok := parseSafariBinaryCookie(page[cookieOffset:])
		if ok {
			cookies = append(cookies, cookie)
		}
	}
	return cookies, nil
}

func parseSafariBinaryCookie(raw []byte) (domainCookie, bool) {
	if len(raw) < 32 {
		return domainCookie{}, false
	}

	size := int(binary.LittleEndian.Uint32(raw[0:4]))
	if size <= 0 || size > len(raw) {
		return domainCookie{}, false
	}
	raw = raw[:size]

	domain, ok := readSafariCString(raw, int(binary.LittleEndian.Uint32(raw[16:20])))
	if !ok {
		return domainCookie{}, false
	}
	name, ok := readSafariCString(raw, int(binary.LittleEndian.Uint32(raw[20:24])))
	if !ok {
		return domainCookie{}, false
	}
	value, ok := readSafariCString(raw, int(binary.LittleEndian.Uint32(raw[28:32])))
	if !ok {
		return domainCookie{}, false
	}

	cookie := domainCookie{
		domain: normalizeTwitterCookieDomain(domain),
		name:   name,
		value:  value,
	}
	if !isTwitterSessionCookie(cookie) {
		return domainCookie{}, false
	}
	return cookie, true
}

func readSafariCString(raw []byte, offset int) (string, bool) {
	if offset <= 0 || offset >= len(raw) {
		return "", false
	}
	end := offset
	for end < len(raw) && raw[end] != 0 {
		end++
	}
	if end == offset {
		return "", false
	}
	return string(raw[offset:end]), true
}

func normalizeTwitterCookieDomain(domain string) string {
	normalized := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(domain)), ".")
	switch {
	case normalized == "x.com" || strings.HasSuffix(normalized, ".x.com"):
		return ".x.com"
	case normalized == "twitter.com" || strings.HasSuffix(normalized, ".twitter.com"):
		return ".twitter.com"
	default:
		return domain
	}
}

func isTwitterSessionCookie(cookie domainCookie) bool {
	if cookie.name != "auth_token" && cookie.name != "ct0" {
		return false
	}
	domain := strings.TrimPrefix(strings.ToLower(cookie.domain), ".")
	return domain == "x.com" || domain == "twitter.com"
}
