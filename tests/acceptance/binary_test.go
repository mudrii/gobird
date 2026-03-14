//go:build acceptance

package acceptance_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

var (
	buildBinaryOnce sync.Once
	builtBinaryPath string
	buildBinaryErr  error
)

func repoRoot() string {
	return filepath.Join("..", "..")
}

func buildBinary(t *testing.T) string {
	t.Helper()

	buildBinaryOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "gobird-binary-*")
		if err != nil {
			buildBinaryErr = err
			return
		}
		builtBinaryPath = filepath.Join(tmpDir, "gobird")

		cmd := exec.Command("go", "build", "-o", builtBinaryPath, "./cmd/gobird")
		cmd.Dir = repoRoot()
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err != nil {
			buildBinaryErr = fmt.Errorf("%w: %s", err, string(out))
			return
		}
	})

	if buildBinaryErr != nil {
		t.Fatalf("build gobird binary: %v", buildBinaryErr)
	}
	return builtBinaryPath
}

func runBinary(t *testing.T, args ...string) (string, string, int) {
	t.Helper()

	return runBinaryWithEnv(t, os.Environ(), args...)
}

func runBinaryWithEnv(t *testing.T, env []string, args ...string) (string, string, int) {
	t.Helper()

	cmd := exec.Command(buildBinary(t), args...)
	cmd.Dir = repoRoot()
	cmd.Env = env
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return stdout.String(), stderr.String(), exitErr.ExitCode()
	}
	t.Fatalf("run binary: %v", err)
	return "", "", 0
}

func filteredEnv(keys ...string) []string {
	skip := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		skip[key] = struct{}{}
	}
	env := make([]string, 0, len(os.Environ()))
	for _, kv := range os.Environ() {
		name := strings.SplitN(kv, "=", 2)[0]
		if _, ok := skip[name]; ok {
			continue
		}
		env = append(env, kv)
	}
	return env
}

func TestBinaryVersion(t *testing.T) {
	stdout, stderr, code := runBinary(t, "--version")
	if code != 0 {
		t.Fatalf("unexpected exit code %d, stderr=%q", code, stderr)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Fatalf("expected version output, got stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestBinaryHelp(t *testing.T) {
	stdout, stderr, code := runBinary(t, "--help")
	if code != 0 {
		t.Fatalf("unexpected exit code %d, stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "gobird") {
		t.Fatalf("help output missing gobird: %q", stdout)
	}
}

func TestBinaryUnknownCommandExitCode(t *testing.T) {
	_, stderr, code := runBinary(t, "no-such-command")
	if code != 2 {
		t.Fatalf("unknown command exit code = %d, stderr=%q", code, stderr)
	}
}

func TestBinaryNegativeCountExitCode(t *testing.T) {
	_, stderr, code := runBinary(t, "version", "--count", "-1")
	if code != 2 {
		t.Fatalf("negative count exit code = %d, stderr=%q", code, stderr)
	}
}

func TestBinaryInvalidReadExitCode(t *testing.T) {
	_, stderr, code := runBinary(t, "read", "not-a-tweet-url")
	if code != 2 {
		t.Fatalf("invalid read exit code = %d, stderr=%q", code, stderr)
	}
}

func TestBinaryCheckNoCredentialsExitCode(t *testing.T) {
	env := filteredEnv(
		"AUTH_TOKEN",
		"TWITTER_AUTH_TOKEN",
		"CT0",
		"TWITTER_CT0",
		"BIRD_CONFIG",
		"CHROME_SAFE_STORAGE_PASSWORD",
		"HOME",
	)
	env = append(env, "HOME="+t.TempDir())

	_, stderr, code := runBinaryWithEnv(
		t,
		env,
		"check",
		"--config", filepath.Join("tests", "fixtures", "config_minimal.json5"),
	)
	if code != 1 {
		t.Fatalf("no-credentials check exit code = %d, stderr=%q", code, stderr)
	}
	if !strings.Contains(stderr, "no valid credentials found") {
		t.Fatalf("expected credential failure in stderr, got %q", stderr)
	}
}

func TestBinaryChromeOverrideRejectsMalformedCookies(t *testing.T) {
	home := t.TempDir()
	password := "test-password"
	writeChromeCookieDB(t, home, password,
		"not-a-valid-token",
		"bad-ct0",
	)

	env := filteredEnv(
		"AUTH_TOKEN",
		"TWITTER_AUTH_TOKEN",
		"CT0",
		"TWITTER_CT0",
		"BIRD_CONFIG",
		"CHROME_SAFE_STORAGE_PASSWORD",
		"HOME",
	)
	env = append(env,
		"HOME="+home,
		"CHROME_SAFE_STORAGE_PASSWORD="+password,
	)

	_, stderr, code := runBinaryWithEnv(
		t,
		env,
		"check",
		"--browser", "chrome",
		"--chrome-profile", "Default",
	)
	if code != 1 {
		t.Fatalf("malformed chrome cookies exit code = %d, stderr=%q", code, stderr)
	}
	if !strings.Contains(stderr, "invalid credentials from browser") {
		t.Fatalf("expected browser validation failure, got %q", stderr)
	}
}

func writeChromeCookieDB(t *testing.T, home, password, authToken, ct0 string) {
	t.Helper()

	dbDir := filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dbDir, "Cookies")

	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE cookies (host_key TEXT, name TEXT, encrypted_value BLOB)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO cookies (host_key, name, encrypted_value) VALUES (?, ?, ?), (?, ?, ?)`,
		".x.com", "auth_token", encryptChromeCookieForTest(".x.com", authToken, password),
		".x.com", "ct0", encryptChromeCookieForTest(".x.com", ct0, password),
	); err != nil {
		t.Fatal(err)
	}
}

func encryptChromeCookieForTest(host, value, password string) []byte {
	hostHash := sha256.Sum256([]byte(host))
	plaintext := append(hostHash[:], []byte(value)...)
	padLen := aes.BlockSize - (len(plaintext) % aes.BlockSize)
	padded := make([]byte, len(plaintext)+padLen)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}

	key := pbkdf2SHA1ForBinaryTest([]byte(password), []byte("saltysalt"), 1003, 16)
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	iv := []byte("                ")
	dst := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(dst, padded)
	return append([]byte("v10"), dst...)
}

func pbkdf2SHA1ForBinaryTest(password, salt []byte, iter, keyLen int) []byte {
	prf := func(data []byte) []byte {
		h := hmac.New(sha1.New, password)
		h.Write(data)
		return h.Sum(nil)
	}
	result := make([]byte, 0, keyLen)
	for block := uint32(1); len(result) < keyLen; block++ {
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
