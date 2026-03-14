package cli_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/mudrii/gobird/internal/cli"
)

func TestRootHelp(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("root --help: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "gobird") {
		t.Errorf("help output missing 'gobird': %q", out)
	}
}

func TestVersionCmd(t *testing.T) {
	cli.SetBuildInfo("1.2.3", "abc1234")
	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("version: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "1.2.3") || !strings.Contains(out, "abc1234") {
		t.Errorf("version output missing expected values: %q", out)
	}
}

func TestVersionFlag(t *testing.T) {
	cli.SetBuildInfo("2.0.0", "def5678")
	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--version: %v", err)
	}
	out := buf.String()
	if got := strings.TrimSpace(out); got != "2.0.0 (def5678)" {
		t.Fatalf("unexpected --version output: %q", got)
	}
}

func TestRootRejectsTooManyArgs(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"one", "two"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for too many args")
	}
}

func TestExitCode_Nil(t *testing.T) {
	if got := cli.ExitCode(nil); got != 0 {
		t.Errorf("ExitCode(nil) = %d, want 0", got)
	}
}

func TestExitCode_UnknownCommand(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"unknown-subcommand-xyz"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Errorf("ExitCode(unknown command) = %d, want 2 (err: %v)", got, err)
	}
}

func TestExitCode_UnknownFlag(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--unknown-flag-xyz"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Errorf("ExitCode(unknown flag) = %d, want 2 (err: %v)", got, err)
	}
}

func TestExitCode_OtherError(t *testing.T) {
	// "network timeout" doesn't match any usage-error pattern → exit 1
	if got := cli.ExitCode(fmt.Errorf("network timeout")); got != 1 {
		t.Errorf("ExitCode(network timeout) = %d, want 1", got)
	}
}

func TestNewRootCmd_HasAllSubcommands(t *testing.T) {
	expected := []string{
		"version", "tweet", "reply", "read", "replies", "thread",
		"search", "mentions", "home", "bookmarks", "unbookmark",
		"following", "followers", "likes", "whoami", "about",
		"follow", "unfollow", "lists", "list-timeline",
		"news", "trending", "user-tweets", "check", "query-ids",
	}

	cmd := cli.NewRootCmd()
	registered := map[string]bool{}
	for _, sub := range cmd.Commands() {
		registered[sub.Name()] = true
	}

	for _, name := range expected {
		if !registered[name] {
			t.Errorf("subcommand %q is not registered", name)
		}
	}
}

func TestMutuallyExclusiveOutputFlags_JsonAndPlain(t *testing.T) {
	// --json and --plain together should be rejected at PersistentPreRunE time.
	// We verify via ExitCode that the resulting error maps to exit 2.
	err := fmt.Errorf("invalid flags: --json, --json-full, and --plain are mutually exclusive")
	if got := cli.ExitCode(err); got != 2 {
		t.Errorf("ExitCode(mutually exclusive flags) = %d, want 2", got)
	}
}

func TestNegativeLimit_ExitCode(t *testing.T) {
	err := fmt.Errorf("invalid value: --count / --limit must be >= 0, got -5")
	if got := cli.ExitCode(err); got != 2 {
		t.Errorf("ExitCode(negative limit) = %d, want 2", got)
	}
}

// TestPersistentPreRunE_MutuallyExclusiveFlags verifies that passing --json and
// --plain together causes PersistentPreRunE to return an error that maps to exit 2.
func TestPersistentPreRunE_MutuallyExclusiveFlags(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	// Use "version" subcommand so RunE doesn't need auth — PersistentPreRunE
	// runs first and should reject the combination before RunE is reached.
	cmd.SetArgs([]string{"version", "--json", "--plain"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for mutually exclusive --json --plain, got nil")
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Errorf("ExitCode = %d, want 2 (err: %v)", got, err)
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error should mention 'mutually exclusive': %v", err)
	}
}

// TestPersistentPreRunE_NegativeCount verifies that --count -1 is rejected with exit 2.
func TestPersistentPreRunE_NegativeCount(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"version", "--count", "-1"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for negative --count, got nil")
	}
	if got := cli.ExitCode(err); got != 2 {
		t.Errorf("ExitCode = %d, want 2 (err: %v)", got, err)
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("error should contain 'invalid': %v", err)
	}
}

// TestRootCmd_ShorthandRead verifies that `gobird <invalid_url>` dispatches to
// the read subcommand. We use an invalid URL so the command fails at ID parsing
// before any network call, confirming dispatch happened (not an auth error).
func TestRootCmd_ShorthandRead(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"not-a-tweet-url"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid tweet URL")
	}
	// The error should be the parse error from read, not an auth error.
	if !strings.Contains(err.Error(), "invalid tweet ID or URL") {
		t.Errorf("expected 'invalid tweet ID or URL' error, got: %v", err)
	}
}

// TestVersionCmd_Output verifies that `gobird version` prints the version string.
func TestVersionCmd_Output(t *testing.T) {
	cli.SetBuildInfo("3.0.0", "fff9999")
	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("version: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "3.0.0") {
		t.Errorf("version output missing version string: %q", out)
	}
	if !strings.Contains(out, "fff9999") {
		t.Errorf("version output missing git SHA: %q", out)
	}
}
