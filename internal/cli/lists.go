package cli

import (
	"fmt"

	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
	"github.com/spf13/cobra"
)

func newListsCmd() *cobra.Command {
	var memberships bool

	cmd := &cobra.Command{
		Use:   "lists",
		Short: "List owned lists (or memberships with --memberships)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			var result *types.ListResult
			if memberships {
				result, err = c.GetListMemberships(cmd.Context(), nil)
			} else {
				result, err = c.GetOwnedLists(cmd.Context(), nil)
			}
			if err != nil {
				return err
			}
			if globalFlags.jsonOutput || globalFlags.jsonFull {
				return output.PrintJSON(cmd.OutOrStdout(), result.Items)
			}
			fmtOpts := currentFormatOptions()
			for _, l := range result.Items {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), output.FormatList(l, fmtOpts)); err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&memberships, "memberships", false, "Show lists you are a member of")

	return cmd
}

func newListTimelineCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "list-timeline <list-id-or-url>",
		Short: "Fetch timeline for a list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			listID := parsing.ExtractListID(args[0])
			if listID == "" {
				return fmt.Errorf("invalid list ID or URL: %q", args[0])
			}
			opts := &types.FetchOptions{Limit: globalFlags.limit, IncludeRaw: globalFlags.jsonFull, QuoteDepth: resolveQuoteDepthFromCommand()}
			result, err := c.GetListTimeline(cmd.Context(), listID, opts)
			if err != nil {
				return err
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

	return cmd
}
