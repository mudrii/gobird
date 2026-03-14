package auth

import (
	"database/sql"
	"encoding/binary"
	"os"
	"path/filepath"
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
