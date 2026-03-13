// Package main is the entry point for the gobird CLI.
package main

import (
	"fmt"
	"os"

	"github.com/mudrii/gobird/internal/cli"
)

// version and gitSHA are injected via ldflags at build time.
var (
	version = "dev"
	gitSHA  = "unknown"
)

func main() {
	cli.SetBuildInfo(version, gitSHA)
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
