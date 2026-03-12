// Package cli implements the bird command-line interface.
package cli

import (
	"github.com/spf13/cobra"
)

var (
	buildVersion = "dev"
	buildGitSHA  = "unknown"
)

// globalFlags holds persistent flag values shared by all subcommands.
var globalFlags struct {
	authToken  string
	ct0        string
	browser    string
	configPath string
	jsonOutput bool
	jsonFull   bool
	plain      bool
	noColor    bool
	noEmoji    bool
	limit      int
	maxPages   int
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
	pf.IntVar(&globalFlags.limit, "limit", 0, "Maximum number of items to fetch")
	pf.IntVar(&globalFlags.maxPages, "max-pages", 0, "Maximum number of pages to fetch")

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
	root.AddCommand(newCheckCmd())
	root.AddCommand(newQueryIDsCmd())

	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Printf("bird %s (%s)\n", buildVersion, buildGitSHA)
			return nil
		},
	}
}
