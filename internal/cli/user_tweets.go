package cli

import (
	"fmt"

	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
	"github.com/spf13/cobra"
)

func newUserTweetsCmd() *cobra.Command {

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
					Limit:      globalFlags.limit,
					IncludeRaw: globalFlags.jsonFull,
					QuoteDepth: resolveQuoteDepthFromCommand(),
				},
			})
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
