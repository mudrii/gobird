// Package main is the entry point for the gobird CLI.
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/mudrii/gobird/internal/cli"
)

// version and gitSHA are injected via ldflags at build time.
// When installed via "go install", they fall back to debug.ReadBuildInfo.
var (
	version = ""
	gitSHA  = ""
)

func init() {
	if version != "" && gitSHA != "" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if version == "" {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		} else {
			version = "dev"
		}
	}
	if gitSHA == "" {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && len(s.Value) >= 7 {
				gitSHA = s.Value[:7]
				return
			}
		}
		gitSHA = "unknown"
	}
}

func main() {
	cli.SetBuildInfo(version, gitSHA)
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
