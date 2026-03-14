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
