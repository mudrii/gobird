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
	if !strings.Contains(out, "bird") {
		t.Errorf("help output missing 'bird': %q", out)
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
