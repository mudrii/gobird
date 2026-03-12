package cli

import (
	"context"
	"fmt"

	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/types"
	"github.com/spf13/cobra"
)

func newFollowingCmd() *cobra.Command {
	var userHandle string
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
			ctx := context.Background()
			var userID string
			if userHandle != "" {
				userID, err = c.GetUserIDByUsername(ctx, stripAtPrefix(userHandle))
				if err != nil {
					return err
				}
			} else {
				u, err := c.GetCurrentUser(ctx)
				if err != nil {
					return err
				}
				userID = u.ID
			}
			opts := &types.FetchOptions{Limit: limit}
			result, err := c.GetFollowing(ctx, userID, opts)
			if err != nil {
				return err
			}
			if asJSON {
				return output.PrintJSON(cmd.OutOrStdout(), result.Items)
			}
			fmtOpts := output.FormatOptions{}
			for _, u := range result.Items {
				fmt.Fprintln(cmd.OutOrStdout(), output.FormatUser(u, fmtOpts))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&userHandle, "user", "", "Twitter handle (e.g. @jack)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of users to fetch")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")

	return cmd
}

func newFollowersCmd() *cobra.Command {
	var userHandle string
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
			ctx := context.Background()
			var userID string
			var fetchErr error
			if userHandle != "" {
				userID, fetchErr = c.GetUserIDByUsername(ctx, stripAtPrefix(userHandle))
				if fetchErr != nil {
					return fetchErr
				}
			} else {
				u, fetchErr := c.GetCurrentUser(ctx)
				if fetchErr != nil {
					return fetchErr
				}
				userID = u.ID
			}
			opts := &types.FetchOptions{Limit: limit}
			result, err := c.GetFollowers(ctx, userID, opts)
			if err != nil {
				return err
			}
			if asJSON {
				return output.PrintJSON(cmd.OutOrStdout(), result.Items)
			}
			fmtOpts := output.FormatOptions{}
			for _, u := range result.Items {
				fmt.Fprintln(cmd.OutOrStdout(), output.FormatUser(u, fmtOpts))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&userHandle, "user", "", "Twitter handle (e.g. @jack)")
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
			ctx := context.Background()
			opts := &types.FetchOptions{Limit: limit}
			result := c.GetLikes(ctx, opts)
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
			ctx := context.Background()
			u, err := c.GetCurrentUser(ctx)
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
			ctx := context.Background()
			u, err := c.GetUserAboutAccount(ctx, handle)
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
		Use:   "follow <@handle>",
		Short: "Follow a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			ctx := context.Background()
			handle := stripAtPrefix(args[0])
			userID, err := c.GetUserIDByUsername(ctx, handle)
			if err != nil {
				return err
			}
			if err := c.Follow(ctx, userID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "followed @%s\n", handle)
			return nil
		},
	}
}

func newUnfollowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unfollow <@handle>",
		Short: "Unfollow a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			ctx := context.Background()
			handle := stripAtPrefix(args[0])
			userID, err := c.GetUserIDByUsername(ctx, handle)
			if err != nil {
				return err
			}
			if err := c.Unfollow(ctx, userID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "unfollowed @%s\n", handle)
			return nil
		},
	}
}
