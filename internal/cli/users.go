package cli

import (
	"fmt"
	"strings"

	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/types"
	"github.com/spf13/cobra"
)

func newFollowingCmd() *cobra.Command {
	var userIDFlag string
	var limit int
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "following",
		Short: "List users the given account follows",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			var userID string
			if userIDFlag != "" {
				if strings.HasPrefix(userIDFlag, "@") {
					return fmt.Errorf("--user must be a numeric user ID, got handle %q", userIDFlag)
				}
				userID = userIDFlag
			} else {
				u, err := c.GetCurrentUser(cmd.Context())
				if err != nil {
					return err
				}
				userID = u.ID
			}
			opts := &types.FetchOptions{Limit: limit, IncludeRaw: globalFlags.jsonFull, QuoteDepth: resolveQuoteDepthFromCommand()}
			result, err := c.GetFollowing(cmd.Context(), userID, opts)
			if err != nil {
				return err
			}
			if asJSON || globalFlags.jsonFull {
				return output.PrintJSON(cmd.OutOrStdout(), result.Items)
			}
			fmtOpts := currentFormatOptions()
			for _, u := range result.Items {
				fmt.Fprintln(cmd.OutOrStdout(), output.FormatUser(u, fmtOpts))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&userIDFlag, "user", "", "Numeric Twitter user ID")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of users to fetch")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")

	return cmd
}

func newFollowersCmd() *cobra.Command {
	var userIDFlag string
	var limit int
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "followers",
		Short: "List followers of the given account",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			var userID string
			if userIDFlag != "" {
				if strings.HasPrefix(userIDFlag, "@") {
					return fmt.Errorf("--user must be a numeric user ID, got handle %q", userIDFlag)
				}
				userID = userIDFlag
			} else {
				u, err := c.GetCurrentUser(cmd.Context())
				if err != nil {
					return err
				}
				userID = u.ID
			}
			opts := &types.FetchOptions{Limit: limit, IncludeRaw: globalFlags.jsonFull}
			result, err := c.GetFollowers(cmd.Context(), userID, opts)
			if err != nil {
				return err
			}
			if asJSON || globalFlags.jsonFull {
				return output.PrintJSON(cmd.OutOrStdout(), result.Items)
			}
			fmtOpts := currentFormatOptions()
			for _, u := range result.Items {
				fmt.Fprintln(cmd.OutOrStdout(), output.FormatUser(u, fmtOpts))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&userIDFlag, "user", "", "Numeric Twitter user ID")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of users to fetch")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")

	return cmd
}

func newLikesCmd() *cobra.Command {
	var limit int
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "likes",
		Short: "Fetch tweets liked by the current user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			opts := &types.FetchOptions{Limit: limit, IncludeRaw: globalFlags.jsonFull}
			result := c.GetLikes(cmd.Context(), opts)
			if result.Error != nil {
				return result.Error
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

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Print the currently authenticated user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			u, err := c.GetCurrentUser(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\nUsername: @%s\nName: %s\n", u.ID, u.Username, u.Name)
			return nil
		},
	}
}

func newAboutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "about <@handle>",
		Short: "Show account info for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			handle := stripAtPrefix(args[0])
			u, err := c.GetUserAboutAccount(cmd.Context(), handle)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\nUsername: @%s\nName: %s\nFollowers: %d\nFollowing: %d\nCreated: %s\n",
				u.ID, u.Username, u.Name, u.FollowersCount, u.FollowingCount, u.CreatedAt)
			return nil
		},
	}
}

func newFollowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "follow <@handle-or-id>",
		Short: "Follow a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			input := args[0]
			userID := input
			label := input
			if strings.HasPrefix(input, "@") {
				label = stripAtPrefix(input)
				var err error
				userID, err = c.GetUserIDByUsername(cmd.Context(), label)
				if err != nil {
					return err
				}
			}
			if err := c.Follow(cmd.Context(), userID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "followed %s\n", label)
			return nil
		},
	}
}

func newUnfollowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unfollow <@handle-or-id>",
		Short: "Unfollow a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			input := args[0]
			userID := input
			label := input
			if strings.HasPrefix(input, "@") {
				label = stripAtPrefix(input)
				var err error
				userID, err = c.GetUserIDByUsername(cmd.Context(), label)
				if err != nil {
					return err
				}
			}
			if err := c.Unfollow(cmd.Context(), userID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "unfollowed %s\n", label)
			return nil
		},
	}
}
