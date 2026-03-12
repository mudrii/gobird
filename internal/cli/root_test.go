package cli_test

import (
	"bytes"
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
