// Package cli implements the gobird command-line interface.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	isatty "github.com/mattn/go-isatty"
	"github.com/mudrii/gobird/internal/client"
	"github.com/mudrii/gobird/pkg/bird"
	"github.com/spf13/cobra"
)

var usageErrorPrefixes = []string{
	"unknown command",
	"unknown flag",
	"missing required flag",
	"accepts ",
	"requires ",
	"invalid value",
	"must be a numeric user ID",
	"more --alt values than --media values",
	"mutually exclusive",
	"invalid flags:",
	"invalid tweet ID or URL:",
	"invalid list ID or URL:",
	"invalid handle:",
	"invalid cookie source ",
	"unsupported media type ",
	"media file too large:",
}

type usageError struct {
	err error
}

func (e *usageError) Error() string {
	return e.err.Error()
}

func (e *usageError) Unwrap() error {
	return e.err
}

func markUsageError(err error) error {
	if err == nil {
		return nil
	}
	return &usageError{err: err}
}

func isUsageError(err error) bool {
	var tagged *usageError
	if errors.As(err, &tagged) {
		return true
	}
	msg := err.Error()
	for _, prefix := range usageErrorPrefixes {
		if strings.HasPrefix(msg, prefix) {
			return true
		}
	}
	return false
}

var (
	buildVersion = "dev"
	buildGitSHA  = "unknown"
)

// globalFlags holds persistent flag values shared by all subcommands.
var globalFlags struct {
	authToken        string
	ct0              string
	browser          string
	configPath       string
	jsonOutput       bool
	jsonFull         bool
	plain            bool
	noColor          bool
	noEmoji          bool
	limit            int
	quoteDepth       int
	timeoutMs        int
	cookieTimeoutMs  int
	maxPages         int
	cookieSources    []string
	chromeProfile    string
	chromeProfileDir string
	firefoxProfile   string
	mediaFiles       []string
	altTexts         []string
	version          bool
	quiet            bool
	dryRun           bool
	rateLimit        float64
}

// SetBuildInfo stores version and git SHA injected at link time.
func SetBuildInfo(version, sha string) {
	buildVersion = version
	buildGitSHA = sha
}

// NewRootCmd constructs and returns the root cobra command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "gobird",
		Short:         "Twitter/X CLI and Go client library",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.MaximumNArgs(1),
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if !globalFlags.quiet && isatty.IsTerminal(os.Stderr.Fd()) {
				fmt.Fprintln(os.Stderr, "WARNING: gobird uses X/Twitter's unofficial APIs. Use at your own risk. Your account may be suspended.")
			}
			if err := validateOutputFlags(); err != nil {
				return err
			}
			return validateLimit(globalFlags.limit)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if globalFlags.version {
				return printVersion(cmd)
			}
			if len(args) == 1 {
				return runRead(cmd, args[0])
			}
			return cmd.Help()
		},
	}

	pf := root.PersistentFlags()
	pf.StringVar(&globalFlags.authToken, "auth-token", "", "Twitter auth_token cookie value")
	pf.StringVar(&globalFlags.ct0, "ct0", "", "Twitter ct0 cookie value")
	pf.StringVar(&globalFlags.browser, "browser", "", "Browser to extract cookies from (safari, chrome, firefox)")
	pf.StringVar(&globalFlags.configPath, "config", "", "Config file path")
	pf.BoolVar(&globalFlags.jsonOutput, "json", false, "Output as JSON")
	pf.BoolVar(&globalFlags.jsonFull, "json-full", false, "Output as full JSON (includes raw)")
	pf.BoolVar(&globalFlags.plain, "plain", false, "Plain text output")
	pf.BoolVar(&globalFlags.noColor, "no-color", false, "Disable ANSI color")
	pf.BoolVar(&globalFlags.noEmoji, "no-emoji", false, "Disable emoji")
	pf.IntVarP(&globalFlags.limit, "count", "n", 0, "Maximum number of items to fetch")
	pf.IntVar(&globalFlags.limit, "limit", 0, "Maximum number of items to fetch")
	pf.IntVar(&globalFlags.maxPages, "max-pages", 0, "Maximum number of pages to fetch")
	pf.StringArrayVar(&globalFlags.cookieSources, "cookie-source", nil, "browser cookie source(s): safari, chrome, firefox")
	pf.StringVar(&globalFlags.chromeProfile, "chrome-profile", "", "Chrome profile name")
	pf.StringVar(&globalFlags.chromeProfileDir, "chrome-profile-dir", "", "Chrome/Chromium profile directory or cookie DB path")
	pf.StringVar(&globalFlags.firefoxProfile, "firefox-profile", "", "Firefox profile name")
	pf.IntVar(&globalFlags.cookieTimeoutMs, "cookie-timeout", 0, "Cookie extraction timeout in ms")
	pf.IntVar(&globalFlags.timeoutMs, "timeout", 0, "HTTP request timeout in ms")
	pf.IntVar(&globalFlags.quoteDepth, "quote-depth", -1, "Quoted tweet expansion depth")
	pf.StringArrayVar(&globalFlags.mediaFiles, "media", nil, "media file path(s) to attach")
	pf.StringArrayVar(&globalFlags.altTexts, "alt", nil, "alt text for the corresponding media file")
	pf.BoolVar(&globalFlags.version, "version", false, "Print version information")
	pf.BoolVarP(&globalFlags.quiet, "quiet", "q", false, "Suppress the startup warning")
	pf.BoolVar(&globalFlags.dryRun, "dry-run", false, "Preview write operations without making API calls")
	pf.Float64Var(&globalFlags.rateLimit, "rate-limit", 1.0, "Maximum requests per second")
	_ = pf.MarkHidden("limit")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newTweetCmd())
	root.AddCommand(newReplyCmd())
	root.AddCommand(newReadCmd())
	root.AddCommand(newRepliesCmd())
	root.AddCommand(newThreadCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newMentionsCmd())
	root.AddCommand(newHomeCmd())
	root.AddCommand(newBookmarksCmd())
	root.AddCommand(newUnbookmarkCmd())
	root.AddCommand(newFollowingCmd())
	root.AddCommand(newFollowersCmd())
	root.AddCommand(newLikesCmd())
	root.AddCommand(newWhoamiCmd())
	root.AddCommand(newAboutCmd())
	root.AddCommand(newFollowCmd())
	root.AddCommand(newUnfollowCmd())
	root.AddCommand(newListsCmd())
	root.AddCommand(newListTimelineCmd())
	root.AddCommand(newNewsCmd())
	root.AddCommand(newTrendingCmd())
	root.AddCommand(newUserTweetsCmd())
	root.AddCommand(newCheckCmd())
	root.AddCommand(newQueryIDsCmd())

	return root
}

// ExitCode maps CLI failures to the documented exit-code classes.
//
//	0 = success
//	1 = runtime error (API, network, auth)
//	2 = usage error (bad flags, bad arguments)
//	3 = auth failure (401/403)
//	4 = rate limit (429)
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	// Check sentinel errors from the bird package.
	if errors.Is(err, bird.ErrUnauthorized) || errors.Is(err, bird.ErrMissingCredentials) {
		return 3
	}
	if errors.Is(err, bird.ErrRateLimit) {
		return 4
	}
	if status, ok := client.HTTPStatusCode(err); ok {
		switch status {
		case 401, 403:
			return 3
		case 429:
			return 4
		}
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return 1
	}
	switch {
	case isUsageError(err):
		return 2
	default:
		return 1
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return printVersion(cmd) },
	}
}

func printVersion(cmd *cobra.Command) error {
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s (%s)\n", buildVersion, buildGitSHA)
	return err
}
