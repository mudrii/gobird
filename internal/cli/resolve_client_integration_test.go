package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mudrii/gobird/internal/client"
	"github.com/mudrii/gobird/internal/testutil"
)

func cliFixture(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("..", "..", "tests", "fixtures", name)
}

func loadCLIFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(cliFixture(t, name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(data)
}

func withResolvedClient(t *testing.T, c *client.Client, err error) {
	t.Helper()
	prev := resolveClientFunc
	resolveClientFunc = func() (*client.Client, error) { return c, err }
	t.Cleanup(func() {
		resolveClientFunc = prev
	})
}

func TestCheckCmd_WithResolvedClientSuccess(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, loadCLIFixture(t, "current_user_response.json")))
	t.Cleanup(srv.Close)

	withResolvedClient(t, client.New("fake-auth", "fake-ct0", &client.Options{
		HTTPClient: testutil.NewHTTPClientForServer(srv),
	}), nil)

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"check"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "OK: @") {
		t.Fatalf("unexpected check output: %q", got)
	}
}

func TestReadCmd_WithResolvedClientJSON(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(200, loadCLIFixture(t, "tweet_detail_response.json")))
	t.Cleanup(srv.Close)

	withResolvedClient(t, client.New("fake-auth", "fake-ct0", &client.Options{
		HTTPClient: testutil.NewHTTPClientForServer(srv),
		QueryIDCache: map[string]string{
			"TweetDetail": "testQID",
		},
	}), nil)

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"read", "https://x.com/example/status/1234567890123456789", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, `"id": "3001"`) {
		t.Fatalf("unexpected read json output: %q", got)
	}
}

func TestCheckCmd_WithResolvedClientAuthFailure(t *testing.T) {
	srv := testutil.NewTestServer(testutil.StaticHandler(401, loadCLIFixture(t, "error_auth.json")))
	t.Cleanup(srv.Close)

	withResolvedClient(t, client.New("fake-auth", "fake-ct0", &client.Options{
		HTTPClient: testutil.NewHTTPClientForServer(srv),
	}), nil)

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"check"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected auth failure")
	}
	if got := ExitCode(err); got != 3 {
		t.Fatalf("ExitCode(auth failure) = %d, want 3 (err=%v)", got, err)
	}
}
