package cli

import (
	"context"
	"fmt"

	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/parsing"
	"github.com/spf13/cobra"
)

func newTweetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tweet <text>",
		Short: "Post a new tweet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputFlags(); err != nil {
				return err
			}
			text := args[0]

			c, err := resolveClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			mediaIDs, err := uploadGlobalMedia(cmd, c)
			if err != nil {
				return err
			}

			tweetID, err := c.TweetWithMedia(cmd.Context(), text, mediaIDs)
			if err != nil {
				return fmt.Errorf("tweet: %w", err)
			}

			if globalFlags.jsonOutput || globalFlags.jsonFull {
				return output.PrintJSON(cmd.OutOrStdout(), map[string]string{"id": tweetID})
			}
			cmd.Println(tweetID)
			return nil
		},
	}
}

func newReplyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reply <tweet-id-or-url> <text>",
		Short: "Reply to a tweet",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutputFlags(); err != nil {
				return err
			}
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

			mediaIDs, err := uploadGlobalMedia(cmd, c)
			if err != nil {
				return err
			}

			newID, err := c.ReplyWithMedia(cmd.Context(), text, tweetID, mediaIDs)
			if err != nil {
				return fmt.Errorf("reply: %w", err)
			}

			if globalFlags.jsonOutput || globalFlags.jsonFull {
				return output.PrintJSON(cmd.OutOrStdout(), map[string]string{"id": newID})
			}
			cmd.Println(newID)
			return nil
		},
	}
}

func uploadGlobalMedia(cmd *cobra.Command, c mediaUploader) ([]string, error) {
	if len(globalFlags.altTexts) > len(globalFlags.mediaFiles) {
		return nil, fmt.Errorf("more --alt values than --media values")
	}

	ctx := cmd.Context()
	mediaIDs := make([]string, 0, len(globalFlags.mediaFiles))
	for i, path := range globalFlags.mediaFiles {
		data, err := loadMedia(path)
		if err != nil {
			return nil, fmt.Errorf("load media %q: %w", path, err)
		}
		mimeType, err := detectMime(path)
		if err != nil {
			return nil, fmt.Errorf("detect media type %q: %w", path, err)
		}
		altText := ""
		if i < len(globalFlags.altTexts) {
			altText = globalFlags.altTexts[i]
		}
		mediaID, err := c.UploadMedia(ctx, data, mimeType, altText)
		if err != nil {
			return nil, fmt.Errorf("upload media %q: %w", path, err)
		}
		mediaIDs = append(mediaIDs, mediaID)
	}
	return mediaIDs, nil
}

type mediaUploader interface {
	UploadMedia(ctx context.Context, data []byte, mimeType, altText string) (string, error)
}
