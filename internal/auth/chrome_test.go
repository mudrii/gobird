package auth

import (
	"context"
	"crypto/aes"
	"crypto/sha256"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestChromeCookieCandidates_NoHint(t *testing.T) {
	home := "/fakehome"
	got := chromeCookieCandidates(home, "")
	want := []string{
		filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default", "Cookies"),
		filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Profile 1", "Cookies"),
		filepath.Join(home, "Library", "Application Support", "Chromium", "Default", "Cookies"),
	}
	if len(got) != len(want) {
		t.Fatalf("len: want %d, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d]: want %q, got %q", i, want[i], got[i])
		}
	}
}

func TestChromeCookieCandidates_ProfileName(t *testing.T) {
	home := "/fakehome"
	got := chromeCookieCandidates(home, "Profile 2")
	if len(got) < 4 {
		t.Fatalf("expected at least 4 candidates with profile hint, got %d", len(got))
	}
	if got[0] != filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Profile 2", "Cookies") {
		t.Errorf("first candidate: got %q", got[0])
	}
	if got[1] != filepath.Join(home, "Library", "Application Support", "Chromium", "Profile 2", "Cookies") {
		t.Errorf("second candidate: got %q", got[1])
	}
}

func TestChromeCookieCandidates_AbsolutePath(t *testing.T) {
	home := "/fakehome"
	got := chromeCookieCandidates(home, "/custom/path/to/profile")
	if got[0] != "/custom/path/to/profile/Cookies" {
		t.Errorf("first candidate for abs path: got %q", got[0])
	}
}

func TestChromeCookieCandidates_DirectFile(t *testing.T) {
	home := "/fakehome"
	cases := []struct {
		hint string
		want string
	}{
		{"my.sqlite", "my.sqlite"},
		{"/path/to/Cookies", "/path/to/Cookies"},
	}
	for _, tc := range cases {
		got := chromeCookieCandidates(home, tc.hint)
		if got[0] != tc.want {
			t.Errorf("hint %q: first candidate want %q, got %q", tc.hint, tc.want, got[0])
		}
	}
}

func TestDecryptChromeCookie_ShortInput(t *testing.T) {
	got, err := decryptChromeCookie(".x.com", []byte("ab"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ab" {
		t.Errorf("want %q, got %q", "ab", got)
	}
}

func TestDecryptChromeCookie_NoPrefix(t *testing.T) {
	plain := "plainvalue"
	got, err := decryptChromeCookie(".x.com", []byte(plain), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != plain {
		t.Errorf("want %q, got %q", plain, got)
	}
}

func TestDecryptChromeCookie_InvalidCiphertextLength(t *testing.T) {
	key := make([]byte, 16)
	// v10 prefix + 5 bytes (not a multiple of block size)
	enc := append([]byte("v10"), make([]byte, 5)...)
	_, err := decryptChromeCookie(".x.com", enc, key)
	if err == nil {
		t.Fatal("expected error for invalid ciphertext length")
	}
}

func TestDecryptChromeCookie_EmptyAfterPrefix(t *testing.T) {
	key := make([]byte, 16)
	enc := []byte("v10")
	_, err := decryptChromeCookie(".x.com", enc, key)
	if err == nil {
		t.Fatal("expected error for empty ciphertext after prefix")
	}
}

func TestDecryptChromeCookie_ValidV10(t *testing.T) {
	password := []byte("testpassword")
	key := pbkdf2SHA1(password, []byte("saltysalt"), 1003, 16)

	plaintext := []byte("cookievalue12345") // exactly 16 bytes
	// PKCS#7 pad to 32 bytes (next block boundary)
	padLen := aes.BlockSize - (len(plaintext) % aes.BlockSize)
	padded := make([]byte, len(plaintext)+padLen)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	iv := []byte("                ") // 16 spaces
	dst := make([]byte, len(padded))
	// Encrypt with CBC
	for i := 0; i < len(padded); i += aes.BlockSize {
		xored := make([]byte, aes.BlockSize)
		var ivBlock []byte
		if i == 0 {
			ivBlock = iv
		} else {
			ivBlock = dst[i-aes.BlockSize : i]
		}
		for j := 0; j < aes.BlockSize; j++ {
			xored[j] = padded[i+j] ^ ivBlock[j]
		}
		block.Encrypt(dst[i:i+aes.BlockSize], xored)
	}

	enc := append([]byte("v10"), dst...)
	got, err := decryptChromeCookie(".x.com", enc, key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != string(plaintext) {
		t.Errorf("want %q, got %q", string(plaintext), got)
	}
}

func TestDecryptChromeCookie_V11Prefix(t *testing.T) {
	password := []byte("testpw")
	key := pbkdf2SHA1(password, []byte("saltysalt"), 1003, 16)

	plaintext := []byte("hello world!!!!!")
	padLen := aes.BlockSize - (len(plaintext) % aes.BlockSize)
	padded := make([]byte, len(plaintext)+padLen)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}

	block, _ := aes.NewCipher(key)
	iv := []byte("                ")
	dst := make([]byte, len(padded))
	for i := 0; i < len(padded); i += aes.BlockSize {
		xored := make([]byte, aes.BlockSize)
		var ivBlock []byte
		if i == 0 {
			ivBlock = iv
		} else {
			ivBlock = dst[i-aes.BlockSize : i]
		}
		for j := 0; j < aes.BlockSize; j++ {
			xored[j] = padded[i+j] ^ ivBlock[j]
		}
		block.Encrypt(dst[i:i+aes.BlockSize], xored)
	}

	enc := append([]byte("v11"), dst...)
	got, err := decryptChromeCookie(".x.com", enc, key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != string(plaintext) {
		t.Errorf("want %q, got %q", string(plaintext), got)
	}
}

func TestDecryptChromeCookie_StripsHostHashPrefix(t *testing.T) {
	password := []byte("testpassword")
	key := pbkdf2SHA1(password, []byte("saltysalt"), 1003, 16)
	host := ".x.com"
	prefix := sha256.Sum256([]byte(host))
	plaintext := append(prefix[:], []byte("3686ba089daa47ab494946db3f8e873eddb32d74")...)

	padLen := aes.BlockSize - (len(plaintext) % aes.BlockSize)
	padded := make([]byte, len(plaintext)+padLen)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	iv := []byte("                ")
	dst := make([]byte, len(padded))
	for i := 0; i < len(padded); i += aes.BlockSize {
		xored := make([]byte, aes.BlockSize)
		var ivBlock []byte
		if i == 0 {
			ivBlock = iv
		} else {
			ivBlock = dst[i-aes.BlockSize : i]
		}
		for j := 0; j < aes.BlockSize; j++ {
			xored[j] = padded[i+j] ^ ivBlock[j]
		}
		block.Encrypt(dst[i:i+aes.BlockSize], xored)
	}

	enc := append([]byte("v10"), dst...)
	got, err := decryptChromeCookie(host, enc, key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "3686ba089daa47ab494946db3f8e873eddb32d74"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestPbkdf2SHA1_KnownVector(t *testing.T) {
	key := pbkdf2SHA1([]byte("password"), []byte("salt"), 1, 20)
	if len(key) != 20 {
		t.Fatalf("key length: want 20, got %d", len(key))
	}
	// RFC 6070 test vector: PBKDF2-SHA1("password", "salt", 1, 20)
	// = 0c 60 c8 0f 96 1f 0e 71 f3 a9 b5 24 af 60 12 06 2f e0 37 a6
	expected := []byte{
		0x0c, 0x60, 0xc8, 0x0f, 0x96, 0x1f, 0x0e, 0x71,
		0xf3, 0xa9, 0xb5, 0x24, 0xaf, 0x60, 0x12, 0x06,
		0x2f, 0xe0, 0x37, 0xa6,
	}
	for i := range expected {
		if key[i] != expected[i] {
			t.Errorf("byte %d: want %02x, got %02x", i, expected[i], key[i])
		}
	}
}

func TestPbkdf2SHA1_ChromeParams(t *testing.T) {
	key := pbkdf2SHA1([]byte("chromepass"), []byte("saltysalt"), 1003, 16)
	if len(key) != 16 {
		t.Errorf("key length: want 16, got %d", len(key))
	}
}

func TestExtractChrome_NoCookieDB(t *testing.T) {
	dir := t.TempDir()
	_, err := extractChromeFromDir(dir, "")
	if err == nil {
		t.Fatal("expected error when no cookie DB exists")
	}
}

func TestChromeCookieKey_UsesEnvironmentOverride(t *testing.T) {
	prevLookup := chromeKeychainPasswordLookup
	chromeKeychainPasswordLookup = func(ctx context.Context) (string, error) {
		return "", errors.New("should not be called")
	}
	t.Cleanup(func() {
		chromeKeychainPasswordLookup = prevLookup
	})

	t.Setenv(chromeSafeStoragePasswordEnv, "terminal-password")

	key, err := chromeCookieKey(context.Background())
	if err != nil {
		t.Fatalf("chromeCookieKey: %v", err)
	}

	want := chromeCookieKeyFromPassword("terminal-password")
	if string(key) != string(want) {
		t.Fatalf("derived key mismatch")
	}
}

func TestChromeCookieKey_KeychainErrorMentionsOverride(t *testing.T) {
	prevLookup := chromeKeychainPasswordLookup
	chromeKeychainPasswordLookup = func(ctx context.Context) (string, error) {
		return "", errors.New("exit status 36")
	}
	t.Cleanup(func() {
		chromeKeychainPasswordLookup = prevLookup
	})

	t.Setenv(chromeSafeStoragePasswordEnv, "")

	_, err := chromeCookieKey(context.Background())
	if err == nil {
		t.Fatal("expected error when keychain lookup fails")
	}
	if !strings.Contains(err.Error(), chromeSafeStoragePasswordEnv) {
		t.Fatalf("expected override hint in error, got %v", err)
	}
}

// extractChromeFromDir is a test helper that exercises the DB path resolution
// and query logic by creating a temp SQLite DB with the Chrome cookies schema.
func extractChromeFromDir(home, profileHint string) (string, error) {
	candidates := chromeCookieCandidates(home, profileHint)
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", os.ErrNotExist
}

func TestExtractChrome_WithMockDB(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, "Library", "Application Support", "Google", "Chrome", "Default")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dbDir, "Cookies")

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE cookies (
		host_key TEXT, name TEXT, encrypted_value BLOB
	)`)
	if err != nil {
		t.Fatal(err)
	}
	// Insert unencrypted cookies (no v10/v11 prefix means they're returned as-is)
	_, err = db.Exec(`INSERT INTO cookies (host_key, name, encrypted_value) VALUES
		('.x.com', 'auth_token', 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'),
		('.x.com', 'ct0', 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb')`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	candidates := chromeCookieCandidates(dir, "")
	found := false
	for _, c := range candidates {
		if c == dbPath {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected %q in candidates, got %v", dbPath, candidates)
	}
}

// cbcEncrypt is a test helper that AES-CBC encrypts plaintext with PKCS#7 padding.
func cbcEncrypt(t *testing.T, key, plaintext []byte) []byte {
	t.Helper()
	padLen := aes.BlockSize - (len(plaintext) % aes.BlockSize)
	padded := make([]byte, len(plaintext)+padLen)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	iv := []byte("                ") // 16 spaces
	dst := make([]byte, len(padded))
	for i := 0; i < len(padded); i += aes.BlockSize {
		xored := make([]byte, aes.BlockSize)
		var ivBlock []byte
		if i == 0 {
			ivBlock = iv
		} else {
			ivBlock = dst[i-aes.BlockSize : i]
		}
		for j := 0; j < aes.BlockSize; j++ {
			xored[j] = padded[i+j] ^ ivBlock[j]
		}
		block.Encrypt(dst[i:i+aes.BlockSize], xored)
	}
	return dst
}

// cbcEncryptRaw is a test helper that AES-CBC encrypts pre-padded data (no auto-padding).
func cbcEncryptRaw(t *testing.T, key, padded []byte) []byte {
	t.Helper()
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	iv := []byte("                ")
	dst := make([]byte, len(padded))
	for i := 0; i < len(padded); i += aes.BlockSize {
		xored := make([]byte, aes.BlockSize)
		var ivBlock []byte
		if i == 0 {
			ivBlock = iv
		} else {
			ivBlock = dst[i-aes.BlockSize : i]
		}
		for j := 0; j < aes.BlockSize; j++ {
			xored[j] = padded[i+j] ^ ivBlock[j]
		}
		block.Encrypt(dst[i:i+aes.BlockSize], xored)
	}
	return dst
}

func TestDecryptChromeCookie_BadPaddingMiddleBytes(t *testing.T) {
	key := chromeCookieKeyFromPassword("testpassword")
	// Build a 32-byte block where the last byte says padding=4 but one middle
	// padding byte is wrong. Correct PKCS#7 padding of 4 means the last 4 bytes
	// should all be 0x04.
	plain := make([]byte, 32)
	copy(plain, []byte("hello world test"))     // first 16 bytes
	copy(plain[16:], []byte("abcdefghijkl"))    // next 12 bytes of content
	plain[28] = 0x04                             // padding byte 1 (correct)
	plain[29] = 0x04                             // padding byte 2 (correct)
	plain[30] = 0x07                             // padding byte 3 (WRONG — should be 0x04)
	plain[31] = 0x04                             // last byte says pad=4

	ciphertext := cbcEncryptRaw(t, key, plain)
	enc := append([]byte("v10"), ciphertext...)
	_, err := decryptChromeCookie(".x.com", enc, key)
	if err == nil {
		t.Fatal("expected error for bad padding middle byte")
	}
	if !strings.Contains(err.Error(), "invalid padding") {
		t.Fatalf("expected 'invalid padding' error, got: %v", err)
	}
}

func TestDecryptChromeCookie_V11WithHostHashPrefix(t *testing.T) {
	key := chromeCookieKeyFromPassword("testpassword")
	host := ".twitter.com"
	hostHash := sha256.Sum256([]byte(host))
	cookieVal := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	plaintext := append(hostHash[:], []byte(cookieVal)...)

	ciphertext := cbcEncrypt(t, key, plaintext)
	enc := append([]byte("v11"), ciphertext...)

	got, err := decryptChromeCookie(host, enc, key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != cookieVal {
		t.Errorf("want %q, got %q", cookieVal, got)
	}
}

func TestExtractChromeWithContext_Integration(t *testing.T) {
	password := "integration-test-pw"
	key := chromeCookieKeyFromPassword(password)

	// Create a temp Chrome cookie DB with encrypted cookies.
	dir := t.TempDir()
	dbDir := filepath.Join(dir, "Library", "Application Support", "Google", "Chrome", "Default")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dbDir, "Cookies")

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE cookies (
		host_key TEXT, name TEXT, encrypted_value BLOB
	)`)
	if err != nil {
		t.Fatal(err)
	}

	authTokenVal := "a1b2c3d4e5f6a7b8c9d0a1b2c3d4e5f6a7b8c9d0"
	ct0Val := "abcdef1234567890abcdef1234567890ab"

	encAuthToken := append([]byte("v10"), cbcEncrypt(t, key, []byte(authTokenVal))...)
	encCt0 := append([]byte("v10"), cbcEncrypt(t, key, []byte(ct0Val))...)

	_, err = db.Exec(`INSERT INTO cookies (host_key, name, encrypted_value) VALUES (?, ?, ?)`,
		".x.com", "auth_token", encAuthToken)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cookies (host_key, name, encrypted_value) VALUES (?, ?, ?)`,
		".x.com", "ct0", encCt0)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	// Override the keychain lookup to return our test password.
	prevLookup := chromeKeychainPasswordLookup
	chromeKeychainPasswordLookup = func(ctx context.Context) (string, error) {
		return password, nil
	}
	t.Cleanup(func() {
		chromeKeychainPasswordLookup = prevLookup
	})

	// Override UserHomeDir by using the env-based password and profile hint
	// pointing directly to the DB file.
	t.Setenv(chromeSafeStoragePasswordEnv, password)
	result, err := extractChromeWithContext(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("extractChromeWithContext: %v", err)
	}
	if result.AuthToken != authTokenVal {
		t.Errorf("auth_token: want %q, got %q", authTokenVal, result.AuthToken)
	}
	if result.Ct0 != ct0Val {
		t.Errorf("ct0: want %q, got %q", ct0Val, result.Ct0)
	}
	wantHeader := "auth_token=" + authTokenVal + "; ct0=" + ct0Val
	if result.CookieHeader != wantHeader {
		t.Errorf("cookie header: want %q, got %q", wantHeader, result.CookieHeader)
	}
}

func TestExtractChromeWithContext_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := extractChromeWithContext(ctx, "")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestExtractChromeWithContext_MissingCookies(t *testing.T) {
	password := "test-missing"
	key := chromeCookieKeyFromPassword(password)

	dir := t.TempDir()
	dbDir := filepath.Join(dir, "Library", "Application Support", "Google", "Chrome", "Default")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dbDir, "Cookies")

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE cookies (
		host_key TEXT, name TEXT, encrypted_value BLOB
	)`)
	if err != nil {
		t.Fatal(err)
	}
	// Only insert auth_token, not ct0 — should fail.
	authTokenVal := "a1b2c3d4e5f6a7b8c9d0a1b2c3d4e5f6a7b8c9d0"
	encAuthToken := append([]byte("v10"), cbcEncrypt(t, key, []byte(authTokenVal))...)
	_, err = db.Exec(`INSERT INTO cookies (host_key, name, encrypted_value) VALUES (?, ?, ?)`,
		".x.com", "auth_token", encAuthToken)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	t.Setenv(chromeSafeStoragePasswordEnv, password)
	_, err = extractChromeWithContext(context.Background(), dbPath)
	if err == nil {
		t.Fatal("expected error when ct0 cookie is missing")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' in error, got: %v", err)
	}
}

func TestExtractChromeWithContext_V11EncryptedWithHostHash(t *testing.T) {
	password := "v11-hosthash-pw"
	key := chromeCookieKeyFromPassword(password)

	dir := t.TempDir()
	dbDir := filepath.Join(dir, "Library", "Application Support", "Google", "Chrome", "Default")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dbDir, "Cookies")

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE cookies (
		host_key TEXT, name TEXT, encrypted_value BLOB
	)`)
	if err != nil {
		t.Fatal(err)
	}

	host := ".x.com"
	hostHash := sha256.Sum256([]byte(host))
	authTokenVal := "f1e2d3c4b5a6f1e2d3c4b5a6f1e2d3c4b5a6f1e2"
	ct0Val := "0123456789abcdef0123456789abcdef01"

	// v11 prefix + host-hash prepended to plaintext
	authPlain := append(hostHash[:], []byte(authTokenVal)...)
	ct0Plain := append(hostHash[:], []byte(ct0Val)...)

	encAuth := append([]byte("v11"), cbcEncrypt(t, key, authPlain)...)
	encCt0 := append([]byte("v11"), cbcEncrypt(t, key, ct0Plain)...)

	_, err = db.Exec(`INSERT INTO cookies (host_key, name, encrypted_value) VALUES (?, ?, ?)`,
		host, "auth_token", encAuth)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO cookies (host_key, name, encrypted_value) VALUES (?, ?, ?)`,
		host, "ct0", encCt0)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	t.Setenv(chromeSafeStoragePasswordEnv, password)
	result, err := extractChromeWithContext(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("extractChromeWithContext: %v", err)
	}
	if result.AuthToken != authTokenVal {
		t.Errorf("auth_token: want %q, got %q", authTokenVal, result.AuthToken)
	}
	if result.Ct0 != ct0Val {
		t.Errorf("ct0: want %q, got %q", ct0Val, result.Ct0)
	}
}
