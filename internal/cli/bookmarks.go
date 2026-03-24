package cli

import (
	"fmt"
	"os"

	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
	"github.com/spf13/cobra"
)

func newBookmarksCmd() *cobra.Command {
	var folderID string

	cmd := &cobra.Command{
		Use:   "bookmarks",
		Short: "Fetch bookmarks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			var result types.TweetResult
			if folderID != "" {
				opts := &types.BookmarkFolderOptions{
					FetchOptions: types.FetchOptions{Limit: globalFlags.limit, IncludeRaw: globalFlags.jsonFull, QuoteDepth: resolveQuoteDepthFromCommand()},
					FolderID:     folderID,
				}
				result = c.GetBookmarkFolderTimeline(cmd.Context(), opts)
			} else {
				opts := &types.FetchOptions{Limit: globalFlags.limit, IncludeRaw: globalFlags.jsonFull, QuoteDepth: resolveQuoteDepthFromCommand()}
				result = c.GetBookmarks(cmd.Context(), opts)
			}
			if result.Error != nil {
				return result.Error
			}
			if globalFlags.jsonOutput || globalFlags.jsonFull {
				return output.PrintJSON(cmd.OutOrStdout(), result.Items)
			}
			fmtOpts := currentFormatOptions()
			for _, t := range result.Items {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), output.FormatTweet(t, fmtOpts)); err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&folderID, "folder", "", "Bookmark folder ID")

	return cmd
}

func newUnbookmarkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unbookmark <tweet-id-or-url>",
		Short: "Remove a tweet from bookmarks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tweetID := parsing.ExtractTweetID(args[0])
			if tweetID == "" {
				return fmt.Errorf("invalid tweet ID or URL: %q", args[0])
			}
			if globalFlags.dryRun {
				fmt.Fprintf(os.Stderr, "[dry-run] would unbookmark tweet %s\n", tweetID)
				return nil
			}
			c, err := quickClient()
			if err != nil {
				return err
			}
			if err := c.Unbookmark(cmd.Context(), tweetID); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "unbookmarked %s\n", tweetID); err != nil {
				return err
			}
			return nil
		},
	}
}
