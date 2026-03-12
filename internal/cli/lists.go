package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/types"
	"github.com/spf13/cobra"
)

func newListsCmd() *cobra.Command {
	var memberships bool
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "lists",
		Short: "List owned lists (or memberships with --memberships)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			ctx := context.Background()
			var result *types.ListResult
			if memberships {
				result, err = c.GetListMemberships(ctx, nil)
			} else {
				result, err = c.GetOwnedLists(ctx, nil)
			}
			if err != nil {
				return err
			}
			if asJSON {
				return output.PrintJSON(cmd.OutOrStdout(), result.Items)
			}
			fmtOpts := output.FormatOptions{}
			for _, l := range result.Items {
				fmt.Fprintln(cmd.OutOrStdout(), output.FormatList(l, fmtOpts))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&memberships, "memberships", false, "Show lists you are a member of")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")

	return cmd
}

func newListTimelineCmd() *cobra.Command {
	var limit int
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "list-timeline <list-id-or-url>",
		Short: "Fetch timeline for a list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			listID := extractListID(args[0])
			ctx := context.Background()
			opts := &types.FetchOptions{Limit: limit}
			result, err := c.GetListTimeline(ctx, listID, opts)
			if err != nil {
				return err
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

	return cmd
}

// extractListID returns the list ID from a URL or raw ID string.
func extractListID(input string) string {
	const listPrefix = "/lists/"
	if idx := strings.LastIndex(input, listPrefix); idx >= 0 {
		rest := input[idx+len(listPrefix):]
		for i, c := range rest {
			if c == '?' || c == '#' || c == '/' {
				return rest[:i]
			}
		}
		return rest
	}
	return input
}
