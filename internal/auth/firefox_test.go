package auth

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// createFirefoxProfile is a test helper that creates a Firefox profile directory
// with a cookies.sqlite database containing the given cookie rows.
func createFirefoxProfile(t *testing.T, base, profileName string, rows [][3]string) {
	t.Helper()
	profDir := filepath.Join(base, profileName)
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
	for _, r := range rows {
		_, err = db.Exec(`INSERT INTO moz_cookies (host, name, value) VALUES (?, ?, ?)`, r[0], r[1], r[2])
		if err != nil {
			t.Fatal(err)
		}
	}
	db.Close()
}

// TestExtractFirefox_MultipleProfilesNoHint verifies that when no profile hint
// is given, all profiles are searched and their cookies are merged. The x.com
// domain from the second profile should win over twitter.com from the first.
func TestExtractFirefox_MultipleProfilesNoHint(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "Library", "Application Support", "Firefox", "Profiles")

	createFirefoxProfile(t, base, "aaa.personal", [][3]string{
		{".twitter.com", "auth_token", "tw_token_a"},
		{".twitter.com", "ct0", "tw_ct0_a"},
	})

	createFirefoxProfile(t, base, "bbb.work", [][3]string{
		{".x.com", "auth_token", "xcom_token_b"},
		{".x.com", "ct0", "xcom_ct0_b"},
	})

	t.Setenv("HOME", dir)

	creds, err := extractFirefox("")
	if err != nil {
		t.Fatalf("extractFirefox: %v", err)
	}
	if creds.AuthToken != "xcom_token_b" {
		t.Errorf("AuthToken: want %q, got %q", "xcom_token_b", creds.AuthToken)
	}
	if creds.Ct0 != "xcom_ct0_b" {
		t.Errorf("Ct0: want %q, got %q", "xcom_ct0_b", creds.Ct0)
	}
}

// TestExtractFirefox_MultipleProfilesNoHint_MergedCookies verifies that cookies
// from different profiles are combined: auth_token from one profile and ct0 from
// another can together form a valid credential set.
func TestExtractFirefox_MultipleProfilesNoHint_MergedCookies(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "Library", "Application Support", "Firefox", "Profiles")

	createFirefoxProfile(t, base, "aaa.first", [][3]string{
		{".x.com", "auth_token", "merged_auth"},
	})

	createFirefoxProfile(t, base, "bbb.second", [][3]string{
		{".x.com", "ct0", "merged_ct0"},
	})

	t.Setenv("HOME", dir)

	creds, err := extractFirefox("")
	if err != nil {
		t.Fatalf("extractFirefox: %v", err)
	}
	if creds.AuthToken != "merged_auth" {
		t.Errorf("AuthToken: want %q, got %q", "merged_auth", creds.AuthToken)
	}
	if creds.Ct0 != "merged_ct0" {
		t.Errorf("Ct0: want %q, got %q", "merged_ct0", creds.Ct0)
	}
}

// TestExtractFirefox_ProfileHintMatchesNone verifies that when a profile hint
// is given but no profile directory matches it, the error reports no cookies found.
func TestExtractFirefox_ProfileHintMatchesNone(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "Library", "Application Support", "Firefox", "Profiles")

	createFirefoxProfile(t, base, "abc.default", [][3]string{
		{".x.com", "auth_token", "should_not_match"},
		{".x.com", "ct0", "should_not_match"},
	})

	t.Setenv("HOME", dir)

	_, err := extractFirefox("nonexistent-profile")
	if err == nil {
		t.Fatal("expected error when hint matches no profiles")
	}
	if !strings.Contains(err.Error(), "no cookies.sqlite found") {
		t.Errorf("error should mention no cookies.sqlite found, got %q", err.Error())
	}
}

// TestReadFirefoxCookiesWithContext_CancelledContext verifies that passing an
// already-cancelled context to readFirefoxCookiesWithContext returns an error
// with the "firefox:" prefix from the query or iteration error wrapping.
func TestReadFirefoxCookiesWithContext_CancelledContext(t *testing.T) {
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
		('.x.com', 'auth_token', 'tok'),
		('.x.com', 'ct0', 'ct')`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = readFirefoxCookiesWithContext(ctx, dbPath)
	if err == nil {
		t.Fatal("expected error for cancelled context on readFirefoxCookiesWithContext")
	}
	if !strings.Contains(err.Error(), "firefox:") {
		t.Errorf("error should have firefox prefix, got %q", err.Error())
	}
}

// TestExtractFirefox_PerProfileErrorAccumulation verifies that when one profile
// has a corrupt database and another has valid cookies, the valid cookies are
// returned. When no valid cookies are found and all profiles errored, the last
// database error is wrapped into the final error message.
func TestExtractFirefox_PerProfileErrorAccumulation(t *testing.T) {
	t.Run("corrupt profile skipped, valid profile used", func(t *testing.T) {
		dir := t.TempDir()
		base := filepath.Join(dir, "Library", "Application Support", "Firefox", "Profiles")

		corruptDir := filepath.Join(base, "aaa.corrupt")
		if err := os.MkdirAll(corruptDir, 0o755); err != nil {
			t.Fatal(err)
		}
		corruptDB, err := sql.Open("sqlite", "file:"+filepath.Join(corruptDir, "cookies.sqlite"))
		if err != nil {
			t.Fatal(err)
		}
		_, err = corruptDB.Exec(`CREATE TABLE wrong_table (id INTEGER)`)
		if err != nil {
			t.Fatal(err)
		}
		corruptDB.Close()

		createFirefoxProfile(t, base, "bbb.valid", [][3]string{
			{".x.com", "auth_token", "good_token"},
			{".x.com", "ct0", "good_ct0"},
		})

		t.Setenv("HOME", dir)

		creds, err := extractFirefox("")
		if err != nil {
			t.Fatalf("extractFirefox: %v", err)
		}
		if creds.AuthToken != "good_token" {
			t.Errorf("AuthToken: want %q, got %q", "good_token", creds.AuthToken)
		}
		if creds.Ct0 != "good_ct0" {
			t.Errorf("Ct0: want %q, got %q", "good_ct0", creds.Ct0)
		}
	})

	t.Run("all profiles corrupt wraps last db error", func(t *testing.T) {
		dir := t.TempDir()
		base := filepath.Join(dir, "Library", "Application Support", "Firefox", "Profiles")

		for _, name := range []string{"aaa.bad1", "bbb.bad2"} {
			profDir := filepath.Join(base, name)
			if err := os.MkdirAll(profDir, 0o755); err != nil {
				t.Fatal(err)
			}
			db, err := sql.Open("sqlite", "file:"+filepath.Join(profDir, "cookies.sqlite"))
			if err != nil {
				t.Fatal(err)
			}
			_, err = db.Exec(`CREATE TABLE wrong_table (id INTEGER)`)
			if err != nil {
				t.Fatal(err)
			}
			db.Close()
		}

		t.Setenv("HOME", dir)

		_, err := extractFirefox("")
		if err == nil {
			t.Fatal("expected error when all profiles are corrupt")
		}
		if !strings.Contains(err.Error(), "auth_token or ct0 not found") {
			t.Errorf("error should mention missing cookies, got %q", err.Error())
		}
		if !strings.Contains(err.Error(), "query cookies") {
			t.Errorf("error should wrap last DB error with 'query cookies', got %q", err.Error())
		}
	})
}

// TestExtractFirefoxWithContext_CancelledBeforeStart verifies that when the
// context is already cancelled before extractFirefoxWithContext begins, it
// returns the context error immediately without touching the filesystem.
func TestExtractFirefoxWithContext_CancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := extractFirefoxWithContext(ctx, "")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want context.Canceled, got %v", err)
	}
}

// TestExtractFirefoxWithContext_CancelledBetweenProfiles verifies that context
// cancellation between profile iterations is detected, stopping the loop early.
func TestExtractFirefoxWithContext_CancelledBetweenProfiles(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "Library", "Application Support", "Firefox", "Profiles")

	createFirefoxProfile(t, base, "aaa.first", [][3]string{
		{".x.com", "auth_token", "tok1"},
		{".x.com", "ct0", "ct01"},
	})
	createFirefoxProfile(t, base, "bbb.second", [][3]string{
		{".x.com", "auth_token", "tok2"},
		{".x.com", "ct0", "ct02"},
	})

	t.Setenv("HOME", dir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := extractFirefoxWithContext(ctx, "")
	if err == nil {
		t.Fatal("expected error for cancelled context during profile iteration")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want context.Canceled, got %v", err)
	}
}
