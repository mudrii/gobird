// Package cli implements the bird command-line interface.
package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

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
}

// SetBuildInfo stores version and git SHA injected at link time.
func SetBuildInfo(version, sha string) {
	buildVersion = version
	buildGitSHA = sha
}

// NewRootCmd constructs and returns the root cobra command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "bird",
		Short:         "Twitter/X CLI and Go client library",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.MaximumNArgs(1),
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
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
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unknown command"),
		strings.Contains(msg, "unknown flag"),
		strings.Contains(msg, "accepts"),
		strings.Contains(msg, "requires"),
		strings.Contains(msg, "invalid"),
		strings.Contains(msg, "missing"):
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
