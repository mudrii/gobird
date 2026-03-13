package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mudrii/gobird/internal/cli"
)

func TestSubcommandHelp(t *testing.T) {
	subcmds := []string{
		"tweet", "reply", "read", "replies", "thread",
		"search", "mentions", "home", "bookmarks", "unbookmark",
		"following", "followers", "likes", "whoami", "about",
		"follow", "unfollow", "lists", "list-timeline",
		"news", "trending", "user-tweets", "check", "query-ids",
	}
	for _, name := range subcmds {
		t.Run(name, func(t *testing.T) {
			cmd := cli.NewRootCmd()
			buf := &bytes.Buffer{}
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs([]string{name, "--help"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("%s --help: %v", name, err)
			}
			out := buf.String()
			if !strings.Contains(out, name) {
				t.Errorf("%s help output missing command name", name)
			}
		})
	}
}

func TestSubcommandArgValidation(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"tweet requires text", []string{"tweet"}, true},
		{"reply requires two args", []string{"reply", "only-one"}, true},
		{"read requires one arg", []string{"read"}, true},
		{"replies requires one arg", []string{"replies"}, true},
		{"thread requires one arg", []string{"thread"}, true},
		{"search requires query", []string{"search"}, true},
		{"about requires handle", []string{"about"}, true},
		{"follow requires arg", []string{"follow"}, true},
		{"unfollow requires arg", []string{"unfollow"}, true},
		{"unbookmark requires arg", []string{"unbookmark"}, true},
		{"bookmarks takes no args", []string{"bookmarks", "extra"}, true},
		{"home takes no args", []string{"home", "extra"}, true},
		{"check takes no args", []string{"check", "extra"}, true},
		{"whoami takes no args", []string{"whoami", "extra"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := cli.NewRootCmd()
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if tc.wantErr && err == nil {
				t.Errorf("expected error for args %v", tc.args)
			}
		})
	}
}

func TestPersistentFlags_Defaults(t *testing.T) {
	cmd := cli.NewRootCmd()
	pf := cmd.PersistentFlags()

	cases := []struct {
		flag     string
		wantDef  string
	}{
		{"auth-token", ""},
		{"ct0", ""},
		{"browser", ""},
		{"config", ""},
		{"chrome-profile", ""},
		{"firefox-profile", ""},
	}
	for _, tc := range cases {
		f := pf.Lookup(tc.flag)
		if f == nil {
			t.Errorf("persistent flag %q not found", tc.flag)
			continue
		}
		if f.DefValue != tc.wantDef {
			t.Errorf("flag %q default: want %q, got %q", tc.flag, tc.wantDef, f.DefValue)
		}
	}
}

func TestPersistentFlags_BoolDefaults(t *testing.T) {
	cmd := cli.NewRootCmd()
	pf := cmd.PersistentFlags()

	boolFlags := []string{"json", "json-full", "plain", "no-color", "no-emoji", "version"}
	for _, name := range boolFlags {
		f := pf.Lookup(name)
		if f == nil {
			t.Errorf("persistent flag %q not found", name)
			continue
		}
		if f.DefValue != "false" {
			t.Errorf("bool flag %q default: want %q, got %q", name, "false", f.DefValue)
		}
	}
}

func TestPersistentFlags_IntDefaults(t *testing.T) {
	cmd := cli.NewRootCmd()
	pf := cmd.PersistentFlags()

	intFlags := []struct {
		name    string
		wantDef string
	}{
		{"count", "0"},
		{"max-pages", "0"},
		{"cookie-timeout", "0"},
		{"timeout", "0"},
		{"quote-depth", "-1"},
	}
	for _, tc := range intFlags {
		f := pf.Lookup(tc.name)
		if f == nil {
			t.Errorf("persistent flag %q not found", tc.name)
			continue
		}
		if f.DefValue != tc.wantDef {
			t.Errorf("int flag %q default: want %q, got %q", tc.name, tc.wantDef, f.DefValue)
		}
	}
}

func TestCountShorthand(t *testing.T) {
	cmd := cli.NewRootCmd()
	pf := cmd.PersistentFlags()
	f := pf.ShorthandLookup("n")
	if f == nil {
		t.Fatal("-n shorthand not registered")
	}
	if f.Name != "count" {
		t.Errorf("-n maps to %q, want %q", f.Name, "count")
	}
}

func TestLimitFlagIsHidden(t *testing.T) {
	cmd := cli.NewRootCmd()
	f := cmd.PersistentFlags().Lookup("limit")
	if f == nil {
		t.Fatal("--limit flag not found")
	}
	if !f.Hidden {
		t.Error("--limit should be hidden")
	}
}

func TestExitCode_TableDriven(t *testing.T) {
	cases := []struct {
		msg  string
		want int
	}{
		{"unknown command", 2},
		{"unknown flag", 2},
		{"accepts at most 1 arg", 2},
		{"requires exactly 1 arg", 2},
		{"invalid value", 2},
		{"missing required flag", 2},
		{"connection refused", 1},
		{"timeout exceeded", 1},
		{"rate limited", 1},
	}
	for _, tc := range cases {
		t.Run(tc.msg, func(t *testing.T) {
			err := &simpleError{msg: tc.msg}
			if got := cli.ExitCode(err); got != tc.want {
				t.Errorf("ExitCode(%q) = %d, want %d", tc.msg, got, tc.want)
			}
		})
	}
}

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }

func TestPersistentPreRunE_JsonFullAndPlain(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"version", "--json-full", "--plain"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for --json-full --plain")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error should mention 'mutually exclusive': %v", err)
	}
}

func TestPersistentPreRunE_JsonAndJsonFull(t *testing.T) {
	cmd := cli.NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"version", "--json", "--json-full"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for --json --json-full")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error should mention 'mutually exclusive': %v", err)
	}
}

func TestPersistentPreRunE_ZeroCount(t *testing.T) {
	cli.SetBuildInfo("test", "test")
	cmd := cli.NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"version", "--count", "0"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("--count 0 should be valid: %v", err)
	}
}
