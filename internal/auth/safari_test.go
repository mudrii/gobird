package auth

import (
	"database/sql"
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
