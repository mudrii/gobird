package cli

import (
	"context"
	"fmt"

	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/types"
	"github.com/spf13/cobra"
)

func newBookmarksCmd() *cobra.Command {
	var limit int
	var asJSON bool
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
			ctx := context.Background()
			var result types.TweetResult
			if folderID != "" {
				opts := &types.BookmarkFolderOptions{
					FetchOptions: types.FetchOptions{Limit: limit},
					FolderID:     folderID,
				}
				result = c.GetBookmarkFolderTimeline(ctx, opts)
			} else {
				opts := &types.FetchOptions{Limit: limit}
				result = c.GetBookmarks(ctx, opts)
			}
			if result.Error != nil {
				return result.Error
			}
			if asJSON {
				return output.PrintJSON(cmd.OutOrStdout(), result.Items)
			}
			fmtOpts := output.FormatOptions{}
			for _, t := range result.Items {
				fmt.Fprintln(cmd.OutOrStdout(), output.FormatTweet(t, fmtOpts))
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of tweets to fetch")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&folderID, "folder", "", "Bookmark folder ID")

	return cmd
}

func newUnbookmarkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unbookmark <tweet-id-or-url>",
		Short: "Remove a tweet from bookmarks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			tweetID := extractTweetID(args[0])
			ctx := context.Background()
			if err := c.Unbookmark(ctx, tweetID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "unbookmarked %s\n", tweetID)
			return nil
		},
	}
}

// extractTweetID returns the numeric tweet ID from a URL or raw ID string.
func extractTweetID(input string) string {
	// Handle URLs like https://twitter.com/user/status/1234567890
	const statusPrefix = "/status/"
	if idx := lastIndex(input, statusPrefix); idx >= 0 {
		rest := input[idx+len(statusPrefix):]
		// Trim any trailing query/fragment
		for i, c := range rest {
			if c == '?' || c == '#' || c == '/' {
				return rest[:i]
			}
		}
		return rest
	}
	return input
}

func lastIndex(s, substr string) int {
	last := -1
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			last = i
		}
	}
	return last
}
