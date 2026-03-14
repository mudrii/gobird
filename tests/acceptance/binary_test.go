//go:build acceptance

package acceptance_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
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

	cmd := exec.Command(buildBinary(t), args...)
	cmd.Dir = repoRoot()
	cmd.Env = os.Environ()
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
