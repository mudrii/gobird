package cli

import (
	"fmt"

	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
	"github.com/spf13/cobra"
)

func newUserTweetsCmd() *cobra.Command {
	var limit int
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "user-tweets <@handle>",
		Short: "Fetch tweets from a user timeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			handle := parsing.NormalizeHandle(args[0])
			if handle == "" {
				return fmt.Errorf("invalid handle: %q", args[0])
			}

			c, err := quickClient()
			if err != nil {
				return err
			}
			userID, err := c.GetUserIDByUsername(cmd.Context(), handle)
			if err != nil {
				return err
			}

			result, err := c.GetUserTweets(cmd.Context(), userID, &types.UserTweetsOptions{
				FetchOptions: types.FetchOptions{
					Limit:      limit,
					IncludeRaw: globalFlags.jsonFull,
					QuoteDepth: resolveQuoteDepthFromCommand(),
				},
			})
			if err != nil {
				return err
			}
			if asJSON || globalFlags.jsonFull {
				return output.PrintJSON(cmd.OutOrStdout(), result.Items)
			}
			fmtOpts := currentFormatOptions()
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
