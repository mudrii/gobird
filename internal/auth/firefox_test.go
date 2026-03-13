package auth

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestReadFirefoxCookies_ValidDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cookies.sqlite")

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE moz_cookies (
		host TEXT, name TEXT, value TEXT
	)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO moz_cookies (host, name, value) VALUES
		('.x.com', 'auth_token', 'ff_auth_token_value'),
		('.x.com', 'ct0', 'ff_ct0_value'),
		('.twitter.com', 'auth_token', 'tw_auth_token_value'),
		('.twitter.com', 'ct0', 'tw_ct0_value')`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	cookies, err := readFirefoxCookies(dbPath)
	if err != nil {
		t.Fatalf("readFirefoxCookies: %v", err)
	}
	if len(cookies) != 4 {
		t.Fatalf("want 4 cookies, got %d", len(cookies))
	}
}

func TestReadFirefoxCookies_EmptyDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cookies.sqlite")

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE moz_cookies (
		host TEXT, name TEXT, value TEXT
	)`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	cookies, err := readFirefoxCookies(dbPath)
	if err != nil {
		t.Fatalf("readFirefoxCookies: %v", err)
	}
	if len(cookies) != 0 {
		t.Errorf("want 0 cookies, got %d", len(cookies))
	}
}

func TestReadFirefoxCookies_NonexistentDB(t *testing.T) {
	_, err := readFirefoxCookies("/nonexistent/path/cookies.sqlite")
	if err == nil {
		t.Fatal("expected error for nonexistent DB")
	}
}

func TestReadFirefoxCookies_OnlyTwitterDomain(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cookies.sqlite")

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE moz_cookies (host TEXT, name TEXT, value TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO moz_cookies (host, name, value) VALUES
		('.example.com', 'auth_token', 'should_not_match'),
		('.twitter.com', 'auth_token', 'tw_token'),
		('.twitter.com', 'ct0', 'tw_ct0')`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	cookies, err := readFirefoxCookies(dbPath)
	if err != nil {
		t.Fatalf("readFirefoxCookies: %v", err)
	}
	if len(cookies) != 2 {
		t.Errorf("want 2 cookies (only twitter.com), got %d", len(cookies))
	}
}

func TestExtractFirefox_NoProfileDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	_, err := extractFirefox("")
	if err == nil {
		t.Fatal("expected error when Firefox profile dir doesn't exist")
	}
}

func TestExtractFirefox_EmptyProfiles(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "Library", "Application Support", "Firefox", "Profiles")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", dir)

	_, err := extractFirefox("")
	if err == nil {
		t.Fatal("expected error when no profiles have cookies.sqlite")
	}
}

func TestExtractFirefox_WithProfile(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "Library", "Application Support", "Firefox", "Profiles", "abc123.default")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(profileDir, "cookies.sqlite")
	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE moz_cookies (host TEXT, name TEXT, value TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO moz_cookies (host, name, value) VALUES
		('.x.com', 'auth_token', 'ff_auth'),
		('.x.com', 'ct0', 'ff_ct0')`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	t.Setenv("HOME", dir)
	creds, err := extractFirefox("")
	if err != nil {
		t.Fatalf("extractFirefox: %v", err)
	}
	if creds.AuthToken != "ff_auth" {
		t.Errorf("AuthToken: want %q, got %q", "ff_auth", creds.AuthToken)
	}
	if creds.Ct0 != "ff_ct0" {
		t.Errorf("Ct0: want %q, got %q", "ff_ct0", creds.Ct0)
	}
}

func TestExtractFirefox_ProfileHint(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "Library", "Application Support", "Firefox", "Profiles")

	// Create two profiles
	for _, name := range []string{"abc.default", "xyz.work"} {
		profDir := filepath.Join(base, name)
		if err := os.MkdirAll(profDir, 0o755); err != nil {
			t.Fatal(err)
		}
		dbPath := filepath.Join(profDir, "cookies.sqlite")
		db, err := sql.Open("sqlite", "file:"+dbPath)
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec(`CREATE TABLE moz_cookies (host TEXT, name TEXT, value TEXT)`)
		if err != nil {
			t.Fatal(err)
		}
		token := "token_" + name
		ct0 := "ct0_" + name
		_, err = db.Exec(`INSERT INTO moz_cookies (host, name, value) VALUES
			('.x.com', 'auth_token', ?),
			('.x.com', 'ct0', ?)`, token, ct0)
		if err != nil {
			t.Fatal(err)
		}
		db.Close()
	}

	t.Setenv("HOME", dir)

	creds, err := extractFirefox("work")
	if err != nil {
		t.Fatalf("extractFirefox with hint: %v", err)
	}
	if creds.AuthToken != "token_xyz.work" {
		t.Errorf("AuthToken: want %q, got %q", "token_xyz.work", creds.AuthToken)
	}
}

func TestExtractFirefox_MissingCt0(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "Library", "Application Support", "Firefox", "Profiles", "test.default")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(profileDir, "cookies.sqlite")
	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE moz_cookies (host TEXT, name TEXT, value TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO moz_cookies (host, name, value) VALUES
		('.x.com', 'auth_token', 'only_token')`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	t.Setenv("HOME", dir)
	_, err = extractFirefox("")
	if err == nil {
		t.Fatal("expected error when ct0 is missing")
	}
}
