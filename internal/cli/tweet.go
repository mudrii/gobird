package cli

import (
	"encoding/json"
	"fmt"

	"github.com/mudrii/gobird/internal/parsing"
	"github.com/spf13/cobra"
)

func newTweetCmd() *cobra.Command {
	var mediaFiles []string

	cmd := &cobra.Command{
		Use:   "tweet <text>",
		Short: "Post a new tweet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := args[0]

			c, err := resolveClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			ctx := cmd.Context()

			_ = mediaFiles

			tweetID, err := c.Tweet(ctx, text)
			if err != nil {
				return fmt.Errorf("tweet: %w", err)
			}

			if globalFlags.jsonOutput || globalFlags.jsonFull {
				out, _ := json.MarshalIndent(map[string]string{"id": tweetID}, "", "  ")
				cmd.Println(string(out))
			} else {
				cmd.Println(tweetID)
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&mediaFiles, "media", nil, "media file path(s) to attach")
	return cmd
}

func newReplyCmd() *cobra.Command {
	var mediaFiles []string

	cmd := &cobra.Command{
		Use:   "reply <tweet-id-or-url> <text>",
		Short: "Reply to a tweet",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]
			text := args[1]

			tweetID := parsing.ExtractTweetID(input)
			if tweetID == "" {
				return fmt.Errorf("invalid tweet ID or URL: %q", input)
			}

			c, err := resolveClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			ctx := cmd.Context()

			_ = mediaFiles

			newID, err := c.Reply(ctx, text, tweetID)
			if err != nil {
				return fmt.Errorf("reply: %w", err)
			}

			if globalFlags.jsonOutput || globalFlags.jsonFull {
				out, _ := json.MarshalIndent(map[string]string{"id": newID}, "", "  ")
				cmd.Println(string(out))
			} else {
				cmd.Println(newID)
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&mediaFiles, "media", nil, "media file path(s) to attach")
	return cmd
}
