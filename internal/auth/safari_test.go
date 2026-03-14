package auth

import (
	"context"
	"database/sql"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestExtractSafari_NoDB(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	_, err := extractSafari()
	if err == nil {
		t.Fatal("expected error when Safari cookie DB doesn't exist")
	}
}

func TestExtractSafari_SandboxedPath(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, "Library", "Containers", "com.apple.Safari", "Data", "Library", "Cookies")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dbDir, "Cookies.db")

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE cookies (domain TEXT, name TEXT, value TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cookies (domain, name, value) VALUES
		('.x.com', 'auth_token', 'safari_auth'),
		('.x.com', 'ct0', 'safari_ct0')`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	t.Setenv("HOME", dir)
	creds, err := extractSafari()
	if err != nil {
		t.Fatalf("extractSafari: %v", err)
	}
	if creds.AuthToken != "safari_auth" {
		t.Errorf("AuthToken: want %q, got %q", "safari_auth", creds.AuthToken)
	}
	if creds.Ct0 != "safari_ct0" {
		t.Errorf("Ct0: want %q, got %q", "safari_ct0", creds.Ct0)
	}
}

func TestExtractSafari_SandboxedBinaryCookiesPath(t *testing.T) {
	dir := t.TempDir()
	cookieDir := filepath.Join(dir, "Library", "Containers", "com.apple.Safari", "Data", "Library", "Cookies")
	if err := os.MkdirAll(cookieDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cookiePath := filepath.Join(cookieDir, "Cookies.binarycookies")
	writeSafariBinaryCookiesFile(t, cookiePath, []domainCookie{
		{domain: ".twitter.com", name: "auth_token", value: "tw_auth"},
		{domain: ".twitter.com", name: "ct0", value: "tw_ct0"},
		{domain: "sub.x.com", name: "auth_token", value: "safari_auth"},
		{domain: "sub.x.com", name: "ct0", value: "safari_ct0"},
	})

	t.Setenv("HOME", dir)
	creds, err := extractSafari()
	if err != nil {
		t.Fatalf("extractSafari binarycookies: %v", err)
	}
	if creds.AuthToken != "safari_auth" {
		t.Errorf("AuthToken: want %q, got %q", "safari_auth", creds.AuthToken)
	}
	if creds.Ct0 != "safari_ct0" {
		t.Errorf("Ct0: want %q, got %q", "safari_ct0", creds.Ct0)
	}
}

func TestExtractSafari_FallbackPath(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, "Library", "Cookies")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dbDir, "Cookies.db")

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE cookies (domain TEXT, name TEXT, value TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cookies (domain, name, value) VALUES
		('.twitter.com', 'auth_token', 'safari_tw_auth'),
		('.twitter.com', 'ct0', 'safari_tw_ct0')`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	t.Setenv("HOME", dir)
	creds, err := extractSafari()
	if err != nil {
		t.Fatalf("extractSafari fallback: %v", err)
	}
	if creds.AuthToken != "safari_tw_auth" {
		t.Errorf("AuthToken: want %q, got %q", "safari_tw_auth", creds.AuthToken)
	}
}

func TestExtractSafari_MissingAuthToken(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, "Library", "Cookies")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dbDir, "Cookies.db")

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE cookies (domain TEXT, name TEXT, value TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cookies (domain, name, value) VALUES
		('.x.com', 'ct0', 'only_ct0')`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	t.Setenv("HOME", dir)
	_, err = extractSafari()
	if err == nil {
		t.Fatal("expected error when auth_token is missing")
	}
}

func TestExtractSafari_EmptyDB(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, "Library", "Cookies")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dbDir, "Cookies.db")

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE cookies (domain TEXT, name TEXT, value TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	t.Setenv("HOME", dir)
	_, err = extractSafari()
	if err == nil {
		t.Fatal("expected error when no cookies in DB")
	}
}

func TestExtractSafari_PrefersXComOverTwitter(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, "Library", "Cookies")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dbDir, "Cookies.db")

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE cookies (domain TEXT, name TEXT, value TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cookies (domain, name, value) VALUES
		('.twitter.com', 'auth_token', 'tw_token'),
		('.twitter.com', 'ct0', 'tw_ct0'),
		('.x.com', 'auth_token', 'x_token'),
		('.x.com', 'ct0', 'x_ct0')`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	t.Setenv("HOME", dir)
	creds, err := extractSafari()
	if err != nil {
		t.Fatalf("extractSafari: %v", err)
	}
	if creds.AuthToken != "x_token" {
		t.Errorf("AuthToken: want x.com token, got %q", creds.AuthToken)
	}
	if creds.Ct0 != "x_ct0" {
		t.Errorf("Ct0: want x.com ct0, got %q", creds.Ct0)
	}
}

// --- parseSafariBinaryCookies: corrupt/truncated data ---

func TestParseSafariBinaryCookies_TooShort(t *testing.T) {
	_, err := parseSafariBinaryCookies(context.Background(), []byte("cook"))
	if err == nil {
		t.Fatal("expected error for data shorter than 8 bytes")
	}
	if !strings.Contains(err.Error(), "truncated header") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseSafariBinaryCookies_BadMagic(t *testing.T) {
	data := []byte("baadXXXX")
	_, err := parseSafariBinaryCookies(context.Background(), data)
	if err == nil {
		t.Fatal("expected error for invalid magic")
	}
	if !strings.Contains(err.Error(), "invalid header") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseSafariBinaryCookies_ZeroPages(t *testing.T) {
	data := make([]byte, 8)
	copy(data[:4], []byte("cook"))
	binary.BigEndian.PutUint32(data[4:8], 0)

	cookies, err := parseSafariBinaryCookies(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cookies) != 0 {
		t.Errorf("expected 0 cookies, got %d", len(cookies))
	}
}

func TestParseSafariBinaryCookies_TruncatedPageTable(t *testing.T) {
	data := make([]byte, 8)
	copy(data[:4], []byte("cook"))
	binary.BigEndian.PutUint32(data[4:8], 5) // 5 pages but no room for page offsets

	_, err := parseSafariBinaryCookies(context.Background(), data)
	if err == nil {
		t.Fatal("expected error for truncated page table")
	}
	if !strings.Contains(err.Error(), "truncated page table") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseSafariBinaryCookies_ZeroPageSize(t *testing.T) {
	data := make([]byte, 12)
	copy(data[:4], []byte("cook"))
	binary.BigEndian.PutUint32(data[4:8], 1)
	binary.BigEndian.PutUint32(data[8:12], 0) // page size = 0

	_, err := parseSafariBinaryCookies(context.Background(), data)
	if err == nil {
		t.Fatal("expected error for zero page size")
	}
	if !strings.Contains(err.Error(), "invalid page size") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseSafariBinaryCookies_TruncatedPageData(t *testing.T) {
	data := make([]byte, 12)
	copy(data[:4], []byte("cook"))
	binary.BigEndian.PutUint32(data[4:8], 1)
	binary.BigEndian.PutUint32(data[8:12], 9999) // page size bigger than remaining data

	_, err := parseSafariBinaryCookies(context.Background(), data)
	if err == nil {
		t.Fatal("expected error for truncated page data")
	}
	if !strings.Contains(err.Error(), "truncated page data") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseSafariBinaryCookies_TruncatedCookiePage(t *testing.T) {
	// Build a file with 1 page whose contents are only 8 bytes (less than 12 minimum).
	pageData := make([]byte, 8)
	data := make([]byte, 12+len(pageData))
	copy(data[:4], []byte("cook"))
	binary.BigEndian.PutUint32(data[4:8], 1)
	binary.BigEndian.PutUint32(data[8:12], uint32(len(pageData)))
	copy(data[12:], pageData)

	_, err := parseSafariBinaryCookies(context.Background(), data)
	if err == nil {
		t.Fatal("expected error for truncated cookie page")
	}
	if !strings.Contains(err.Error(), "truncated cookie page") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- readSafariCString edge cases ---

func TestReadSafariCString_ZeroOffset(t *testing.T) {
	raw := []byte("hello\x00")
	_, ok := readSafariCString(raw, 0)
	if ok {
		t.Error("expected false for offset 0")
	}
}

func TestReadSafariCString_NegativeOffset(t *testing.T) {
	raw := []byte("hello\x00")
	_, ok := readSafariCString(raw, -1)
	if ok {
		t.Error("expected false for negative offset")
	}
}

func TestReadSafariCString_OffsetBeyondBuffer(t *testing.T) {
	raw := []byte("hello\x00")
	_, ok := readSafariCString(raw, len(raw)+1)
	if ok {
		t.Error("expected false for offset beyond buffer")
	}
}

func TestReadSafariCString_OffsetAtEnd(t *testing.T) {
	raw := []byte("hello\x00")
	_, ok := readSafariCString(raw, len(raw))
	if ok {
		t.Error("expected false for offset at buffer length")
	}
}

func TestReadSafariCString_EmptyString(t *testing.T) {
	// Null byte right at offset => empty string => returns false.
	raw := []byte{0x00, 0x00}
	_, ok := readSafariCString(raw, 1)
	if ok {
		t.Error("expected false for empty C string")
	}
}

func TestReadSafariCString_NoNullTerminator(t *testing.T) {
	// String runs to end of buffer with no null byte — should still succeed.
	raw := []byte{0x00, 'a', 'b', 'c'}
	s, ok := readSafariCString(raw, 1)
	if !ok {
		t.Fatal("expected success for unterminated string at end of buffer")
	}
	if s != "abc" {
		t.Errorf("want %q, got %q", "abc", s)
	}
}

func TestReadSafariCString_ValidString(t *testing.T) {
	raw := []byte{0xFF, 't', 'e', 's', 't', 0x00, 'x'}
	s, ok := readSafariCString(raw, 1)
	if !ok {
		t.Fatal("expected success")
	}
	if s != "test" {
		t.Errorf("want %q, got %q", "test", s)
	}
}

// --- normalizeTwitterCookieDomain ---

func TestNormalizeTwitterCookieDomain(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"x.com", ".x.com"},
		{".x.com", ".x.com"},
		{"sub.x.com", ".x.com"},
		{".sub.x.com", ".x.com"},
		{"  .X.COM  ", ".x.com"},
		{"twitter.com", ".twitter.com"},
		{".twitter.com", ".twitter.com"},
		{"api.twitter.com", ".twitter.com"},
		{"  .Twitter.Com  ", ".twitter.com"},
		{"example.com", "example.com"},
		{"", ""},
		{" . ", " . "},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeTwitterCookieDomain(tt.input)
			if got != tt.want {
				t.Errorf("normalizeTwitterCookieDomain(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- isTwitterSessionCookie ---

func TestIsTwitterSessionCookie(t *testing.T) {
	tests := []struct {
		name   string
		cookie domainCookie
		want   bool
	}{
		{
			name:   "auth_token on x.com",
			cookie: domainCookie{domain: ".x.com", name: "auth_token", value: "v"},
			want:   true,
		},
		{
			name:   "ct0 on twitter.com",
			cookie: domainCookie{domain: ".twitter.com", name: "ct0", value: "v"},
			want:   true,
		},
		{
			name:   "auth_token without leading dot",
			cookie: domainCookie{domain: "x.com", name: "auth_token", value: "v"},
			want:   true,
		},
		{
			name:   "wrong cookie name",
			cookie: domainCookie{domain: ".x.com", name: "session_id", value: "v"},
			want:   false,
		},
		{
			name:   "wrong domain",
			cookie: domainCookie{domain: ".example.com", name: "auth_token", value: "v"},
			want:   false,
		},
		{
			name:   "subdomain of x.com is not direct match",
			cookie: domainCookie{domain: "sub.x.com", name: "auth_token", value: "v"},
			want:   false,
		},
		{
			name:   "empty cookie",
			cookie: domainCookie{},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTwitterSessionCookie(tt.cookie)
			if got != tt.want {
				t.Errorf("isTwitterSessionCookie(%+v) = %v, want %v", tt.cookie, got, tt.want)
			}
		})
	}
}

// --- Context cancellation for binary parsing ---

func TestParseSafariBinaryCookies_CancelledContext(t *testing.T) {
	// Build a valid binary cookies file with one page containing a cookie.
	record := safariBinaryCookieRecord(".x.com", "auth_token", "tok")
	page := safariBinaryCookiePage([][]byte{record})
	data := make([]byte, 12+len(page))
	copy(data[:4], []byte("cook"))
	binary.BigEndian.PutUint32(data[4:8], 1)
	binary.BigEndian.PutUint32(data[8:12], uint32(len(page)))
	copy(data[12:], page)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := parseSafariBinaryCookies(ctx, data)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !strings.Contains(err.Error(), "canceled") {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestExtractSafariWithContext_CancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := extractSafariWithContext(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !strings.Contains(err.Error(), "canceled") {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

// --- parseSafariBinaryCookies with valid multi-page data ---

func TestParseSafariBinaryCookies_MultiplePages(t *testing.T) {
	page1Records := [][]byte{
		safariBinaryCookieRecord(".x.com", "auth_token", "tok1"),
	}
	page2Records := [][]byte{
		safariBinaryCookieRecord(".x.com", "ct0", "ct0_val"),
	}
	page1 := safariBinaryCookiePage(page1Records)
	page2 := safariBinaryCookiePage(page2Records)

	data := make([]byte, 8+2*4)
	copy(data[:4], []byte("cook"))
	binary.BigEndian.PutUint32(data[4:8], 2)
	binary.BigEndian.PutUint32(data[8:12], uint32(len(page1)))
	binary.BigEndian.PutUint32(data[12:16], uint32(len(page2)))
	data = append(data, page1...)
	data = append(data, page2...)

	cookies, err := parseSafariBinaryCookies(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}
	if cookies[0].name != "auth_token" || cookies[0].value != "tok1" {
		t.Errorf("unexpected first cookie: %+v", cookies[0])
	}
	if cookies[1].name != "ct0" || cookies[1].value != "ct0_val" {
		t.Errorf("unexpected second cookie: %+v", cookies[1])
	}
}

func writeSafariBinaryCookiesFile(t *testing.T, path string, cookies []domainCookie) {
	t.Helper()

	records := make([][]byte, 0, len(cookies))
	for _, cookie := range cookies {
		records = append(records, safariBinaryCookieRecord(cookie.domain, cookie.name, cookie.value))
	}
	page := safariBinaryCookiePage(records)
	data := make([]byte, 12+len(page))
	copy(data[:4], []byte("cook"))
	binary.BigEndian.PutUint32(data[4:8], 1)
	binary.BigEndian.PutUint32(data[8:12], uint32(len(page)))
	copy(data[12:], page)

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func safariBinaryCookiePage(records [][]byte) []byte {
	headerLen := 8 + len(records)*4 + 4
	page := make([]byte, headerLen)
	binary.LittleEndian.PutUint32(page[0:4], 0x100)
	binary.LittleEndian.PutUint32(page[4:8], uint32(len(records)))

	offset := headerLen
	for i, record := range records {
		binary.LittleEndian.PutUint32(page[8+i*4:12+i*4], uint32(offset))
		page = append(page, record...)
		offset += len(record)
	}
	return page
}

func safariBinaryCookieRecord(domain, name, value string) []byte {
	domainBytes := append([]byte(domain), 0)
	nameBytes := append([]byte(name), 0)
	pathBytes := []byte("/\x00")
	valueBytes := append([]byte(value), 0)

	const headerLen = 48
	domainOffset := headerLen
	nameOffset := domainOffset + len(domainBytes)
	pathOffset := nameOffset + len(nameBytes)
	valueOffset := pathOffset + len(pathBytes)
	size := valueOffset + len(valueBytes)

	record := make([]byte, size)
	binary.LittleEndian.PutUint32(record[0:4], uint32(size))
	binary.LittleEndian.PutUint32(record[16:20], uint32(domainOffset))
	binary.LittleEndian.PutUint32(record[20:24], uint32(nameOffset))
	binary.LittleEndian.PutUint32(record[24:28], uint32(pathOffset))
	binary.LittleEndian.PutUint32(record[28:32], uint32(valueOffset))
	copy(record[domainOffset:], domainBytes)
	copy(record[nameOffset:], nameBytes)
	copy(record[pathOffset:], pathBytes)
	copy(record[valueOffset:], valueBytes)
	return record
}
