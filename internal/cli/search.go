package cli

import (
	"fmt"

	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search tweets",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			c, err := resolveClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			opts := &types.SearchOptions{
				FetchOptions: types.FetchOptions{
					Limit:      globalFlags.limit,
					MaxPages:   globalFlags.maxPages,
					QuoteDepth: resolveQuoteDepthFromCommand(),
					IncludeRaw: globalFlags.jsonFull,
				},
			}

			result := c.GetAllSearchResults(cmd.Context(), query, opts)
			if !result.Success && result.Error != nil {
				return fmt.Errorf("search: %w", result.Error)
			}

			if globalFlags.jsonOutput || globalFlags.jsonFull {
				return output.PrintJSON(cmd.OutOrStdout(), result.Items)
			}
			for i := range result.Items {
				printTweet(cmd, &result.Items[i])
				cmd.Println("---")
			}
			return nil
		},
	}
}

func newMentionsCmd() *cobra.Command {
	var user string

	cmd := &cobra.Command{
		Use:   "mentions",
		Short: "Fetch mentions of a user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := resolveClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			handle := user
			if handle == "" {
				cu, err := c.GetCurrentUser(cmd.Context())
				if err != nil {
					return fmt.Errorf("resolve current user: %w", err)
				}
				handle = cu.Username
			}

			query := parsing.MentionsQueryFromUserOption(handle)

			opts := &types.SearchOptions{
				FetchOptions: types.FetchOptions{
					Limit:      globalFlags.limit,
					MaxPages:   globalFlags.maxPages,
					QuoteDepth: resolveQuoteDepthFromCommand(),
					IncludeRaw: globalFlags.jsonFull,
				},
			}

			result := c.GetAllSearchResults(cmd.Context(), query, opts)
			if !result.Success && result.Error != nil {
				return fmt.Errorf("mentions: %w", result.Error)
			}

			if globalFlags.jsonOutput || globalFlags.jsonFull {
				return output.PrintJSON(cmd.OutOrStdout(), result.Items)
			}
			for i := range result.Items {
				printTweet(cmd, &result.Items[i])
				cmd.Println("---")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&user, "user", "", "Twitter handle to fetch mentions for (default: current user)")
	return cmd
}
