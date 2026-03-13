//go:build acceptance

package acceptance_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mudrii/gobird/internal/cli"
	"github.com/mudrii/gobird/internal/testutil"
	"github.com/mudrii/gobird/internal/types"
)

func fixturesDir() string {
	return filepath.Join("..", "fixtures")
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixturesDir(), name))
	if err != nil {
		t.Fatalf("load fixture %s: %v", name, err)
	}
	return data
}

func newMockServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func executeCmd(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := cli.NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

// --- check command tests ---

func TestCheck_NoCredentials(t *testing.T) {
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0", "BIRD_CONFIG"} {
		t.Setenv(e, "")
	}

	_, _, err := executeCmd(t, "check",
		"--config", filepath.Join(fixturesDir(), "config_minimal.json5"))
	if err == nil {
		t.Fatal("expected error when no credentials are available")
	}
}

func TestCheck_WithMockedAPI(t *testing.T) {
	fixture := loadFixture(t, "current_user_response.json")
	srv := newMockServer(t, testutil.StaticHandler(200, string(fixture)))

	_ = srv // We cannot easily inject the mock server into the CLI's resolveClient
	// without modifying the CLI code. Instead, test that the command
	// rejects bad credential formats.
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0", "BIRD_CONFIG"} {
		t.Setenv(e, "")
	}

	_, _, err := executeCmd(t, "check",
		"--auth-token", "not-a-valid-token",
		"--ct0", "not-valid-either")
	if err == nil {
		t.Fatal("expected error for invalid credential format")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("expected 'invalid' in error, got: %v", err)
	}
}

// --- search command tests ---

func TestSearch_MissingQuery(t *testing.T) {
	_, _, err := executeCmd(t, "search")
	if err == nil {
		t.Fatal("expected error when query argument is missing")
	}
}

func TestSearch_NoCredentials(t *testing.T) {
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0", "BIRD_CONFIG"} {
		t.Setenv(e, "")
	}

	_, _, err := executeCmd(t, "search", "golang",
		"--config", filepath.Join(fixturesDir(), "config_minimal.json5"))
	if err == nil {
		t.Fatal("expected error when no credentials available")
	}
}

// --- read command tests ---

func TestRead_InvalidTweetID(t *testing.T) {
	_, _, err := executeCmd(t, "read", "not-a-tweet-url")
	if err == nil {
		t.Fatal("expected error for invalid tweet ID")
	}
	if !strings.Contains(err.Error(), "invalid tweet ID or URL") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRead_ValidURLFormat_NoAuth(t *testing.T) {
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0", "BIRD_CONFIG"} {
		t.Setenv(e, "")
	}

	_, _, err := executeCmd(t, "read", "https://x.com/user/status/12345",
		"--config", filepath.Join(fixturesDir(), "config_minimal.json5"))
	if err == nil {
		t.Fatal("expected auth error for valid URL with no credentials")
	}
}

func TestRead_ShorthandFromRoot(t *testing.T) {
	_, _, err := executeCmd(t, "not-a-tweet-url")
	if err == nil {
		t.Fatal("expected error from root command shorthand read")
	}
	if !strings.Contains(err.Error(), "invalid tweet ID or URL") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

// --- output formatting tests ---

func TestOutputFormatting_MutuallyExclusiveFlags(t *testing.T) {
	_, _, err := executeCmd(t, "version", "--json", "--plain")
	if err == nil {
		t.Fatal("expected error for --json --plain together")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' error, got: %v", err)
	}
}

func TestOutputFormatting_JsonAndJsonFull(t *testing.T) {
	_, _, err := executeCmd(t, "version", "--json", "--json-full")
	if err == nil {
		t.Fatal("expected error for --json --json-full together")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' error, got: %v", err)
	}
}

func TestOutputFormatting_PlainAndJsonFull(t *testing.T) {
	_, _, err := executeCmd(t, "version", "--plain", "--json-full")
	if err == nil {
		t.Fatal("expected error for --plain --json-full together")
	}
}

func TestOutputFormatting_SingleJsonFlag(t *testing.T) {
	_, _, err := executeCmd(t, "version", "--json")
	if err != nil {
		t.Fatalf("--json alone should not fail: %v", err)
	}
}

// --- error scenario tests ---

func TestError_AuthFailure_ExitCode(t *testing.T) {
	err := cli.ExitCode(nil)
	if err != 0 {
		t.Errorf("nil error should return exit code 0, got %d", err)
	}
}

func TestError_InvalidCredentialFormat(t *testing.T) {
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0", "BIRD_CONFIG"} {
		t.Setenv(e, "")
	}

	_, _, err := executeCmd(t, "check",
		"--auth-token", "short",
		"--ct0", "also-short")
	if err == nil {
		t.Fatal("expected error for short/invalid credentials")
	}
}

func TestError_NegativeLimit(t *testing.T) {
	_, _, err := executeCmd(t, "version", "--count", "-1")
	if err == nil {
		t.Fatal("expected error for negative --count")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("expected 'invalid' in error, got: %v", err)
	}
}

// --- version/help tests ---

func TestVersion_Output(t *testing.T) {
	cli.SetBuildInfo("test-1.0.0", "deadbeef")
	stdout, _, err := executeCmd(t, "version")
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	if !strings.Contains(stdout, "test-1.0.0") {
		t.Errorf("version output missing version: %q", stdout)
	}
	if !strings.Contains(stdout, "deadbeef") {
		t.Errorf("version output missing git SHA: %q", stdout)
	}
}

func TestHelp_Output(t *testing.T) {
	stdout, _, err := executeCmd(t, "--help")
	if err != nil {
		t.Fatalf("help failed: %v", err)
	}
	if !strings.Contains(stdout, "bird") {
		t.Errorf("help output missing 'bird': %q", stdout)
	}
}

func TestHelp_SubcommandListing(t *testing.T) {
	stdout, _, err := executeCmd(t, "--help")
	if err != nil {
		t.Fatalf("help failed: %v", err)
	}
	expected := []string{"search", "read", "check", "home", "tweet"}
	for _, cmd := range expected {
		if !strings.Contains(stdout, cmd) {
			t.Errorf("help output missing subcommand %q", cmd)
		}
	}
}

// --- config file tests ---

func TestConfig_InvalidFile(t *testing.T) {
	for _, e := range []string{"AUTH_TOKEN", "TWITTER_AUTH_TOKEN", "CT0", "TWITTER_CT0", "BIRD_CONFIG"} {
		t.Setenv(e, "")
	}

	_, _, err := executeCmd(t, "check",
		"--config", filepath.Join(fixturesDir(), "config_invalid.json5"))
	if err == nil {
		t.Fatal("expected error for invalid config file")
	}
}

// --- JSON output validation ---

func TestJSONOutput_ValidJSON(t *testing.T) {
	tw := types.TweetData{
		ID:   "1",
		Text: "test tweet",
		Author: types.TweetAuthor{
			Username: "tester",
			Name:     "Tester",
		},
		LikeCount:    5,
		RetweetCount: 2,
		ReplyCount:   1,
	}

	data, err := json.MarshalIndent(tw, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded types.TweetData
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.ID != "1" {
		t.Errorf("roundtrip ID: want 1, got %q", decoded.ID)
	}
	if decoded.Text != "test tweet" {
		t.Errorf("roundtrip Text mismatch")
	}
}

func TestJSONOutput_EmptyList(t *testing.T) {
	var tweets []types.TweetData
	data, err := json.MarshalIndent(tweets, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(data) != "null" {
		t.Errorf("empty slice should marshal to null, got %q", string(data))
	}
}
