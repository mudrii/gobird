package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"database/sql"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/mudrii/gobird/internal/types"
)

// extractChrome reads cookies from Chrome's SQLite cookie store on macOS.
// Cookie values are AES-128-CBC encrypted with a key derived from the macOS Keychain.
func extractChrome() (*types.TwitterCookies, error) {
	key, err := chromeCookieKey()
	if err != nil {
		return nil, fmt.Errorf("Chrome: keychain key: %w", err)
	}

	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default", "Cookies"),
		filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Profile 1", "Cookies"),
		filepath.Join(home, "Library", "Application Support", "Chromium", "Default", "Cookies"),
	}

	var dbPath string
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			dbPath = p
			break
		}
	}
	if dbPath == "" {
		return nil, fmt.Errorf("Chrome: cookie database not found")
	}

	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro&immutable=1")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(
		`SELECT host_key, name, encrypted_value FROM cookies WHERE name IN ('auth_token','ct0') AND (host_key LIKE '%x.com' OR host_key LIKE '%twitter.com')`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cookies []domainCookie
	for rows.Next() {
		var host, name string
		var enc []byte
		if err := rows.Scan(&host, &name, &enc); err != nil {
			continue
		}
		val, err := decryptChromeCookie(enc, key)
		if err != nil {
			continue
		}
		cookies = append(cookies, domainCookie{domain: host, name: name, value: val})
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	authToken, ct0 := preferredDomainCookies(cookies)
	if authToken == "" || ct0 == "" {
		return nil, fmt.Errorf("Chrome: auth_token or ct0 not found")
	}
	return &types.TwitterCookies{
		AuthToken:    authToken,
		Ct0:          ct0,
		CookieHeader: buildCookieHeader(authToken, ct0),
	}, nil
}

// chromeCookieKey retrieves the AES key from the macOS Keychain.
func chromeCookieKey() ([]byte, error) {
	out, err := exec.Command(
		"security", "find-generic-password",
		"-w", "-a", "Chrome", "-s", "Chrome Safe Storage",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("keychain access failed: %w", err)
	}
	password := strings.TrimSpace(string(out))
	// Chrome derives a 128-bit (16-byte) AES key using PBKDF2-SHA1 with
	// 1003 iterations and the fixed salt "saltysalt".
	key := pbkdf2SHA1([]byte(password), []byte("saltysalt"), 1003, 16)
	return key, nil
}

// decryptChromeCookie decrypts a Chrome-encrypted cookie value.
// Chrome cookies on macOS start with "v10" or "v11" prefix.
func decryptChromeCookie(enc []byte, key []byte) (string, error) {
	if len(enc) < 3 {
		return string(enc), nil
	}
	prefix := string(enc[:3])
	if prefix != "v10" && prefix != "v11" {
		return string(enc), nil
	}
	enc = enc[3:]
	// IV is 16 space bytes (0x20) for Chrome on macOS.
	iv := []byte("                ")
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	if len(enc) == 0 || len(enc)%aes.BlockSize != 0 {
		return "", fmt.Errorf("invalid ciphertext length")
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	dst := make([]byte, len(enc))
	mode.CryptBlocks(dst, enc)
	// Strip PKCS#7 padding.
	pad := int(dst[len(dst)-1])
	if pad == 0 || pad > aes.BlockSize || pad > len(dst) {
		return "", fmt.Errorf("invalid padding")
	}
	return string(dst[:len(dst)-pad]), nil
}

// pbkdf2SHA1 implements PBKDF2 with HMAC-SHA1 (RFC 2898) using only stdlib.
// Returns keyLen bytes. Supports up to 20*255 output bytes (SHA1 block = 20 bytes).
func pbkdf2SHA1(password, salt []byte, iter, keyLen int) []byte {
	prf := func(data []byte) []byte {
		h := hmac.New(sha1.New, password)
		h.Write(data)
		return h.Sum(nil)
	}
	result := make([]byte, 0, keyLen)
	for block := uint32(1); len(result) < keyLen; block++ {
		// U1 = PRF(password, salt || INT(block))
		s := make([]byte, len(salt)+4)
		copy(s, salt)
		binary.BigEndian.PutUint32(s[len(salt):], block)
		u := prf(s)
		xored := make([]byte, len(u))
		copy(xored, u)
		for i := 1; i < iter; i++ {
			u = prf(u)
			for j := range xored {
				xored[j] ^= u[j]
			}
		}
		result = append(result, xored...)
	}
	return result[:keyLen]
}
