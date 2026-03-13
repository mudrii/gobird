package cli

import (
	"fmt"

	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/parsing"
	"github.com/mudrii/gobird/internal/types"
	"github.com/spf13/cobra"
)

func newReadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "read <tweet-id-or-url>",
		Short: "Read a single tweet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRead(cmd, args[0])
		},
	}
}

func runRead(cmd *cobra.Command, input string) error {
	tweetID := parsing.ExtractTweetID(input)
	if tweetID == "" {
		return fmt.Errorf("invalid tweet ID or URL: %q", input)
	}

	c, err := resolveClient()
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	opts := &types.TweetDetailOptions{
		IncludeRaw: globalFlags.jsonFull,
		QuoteDepth: resolveQuoteDepthFromCommand(),
	}

	tweet, err := c.GetTweet(cmd.Context(), tweetID, opts)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	if globalFlags.jsonOutput || globalFlags.jsonFull {
		return output.PrintJSON(cmd.OutOrStdout(), tweet)
	}
	printTweet(cmd, tweet)
	return nil
}

func newRepliesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "replies <tweet-id-or-url>",
		Short: "Fetch replies to a tweet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tweetID := parsing.ExtractTweetID(args[0])
			if tweetID == "" {
				return fmt.Errorf("invalid tweet ID or URL: %q", args[0])
			}

			c, err := resolveClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			opts := &types.ThreadOptions{
				FetchOptions: types.FetchOptions{
					Limit:      globalFlags.limit,
					MaxPages:   globalFlags.maxPages,
					QuoteDepth: resolveQuoteDepthFromCommand(),
					IncludeRaw: globalFlags.jsonFull,
				},
			}

			result, err := c.GetReplies(cmd.Context(), tweetID, opts)
			if err != nil {
				return fmt.Errorf("replies: %w", err)
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

func newThreadCmd() *cobra.Command {
	var filter string

	cmd := &cobra.Command{
		Use:   "thread <tweet-id-or-url>",
		Short: "Fetch a tweet thread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tweetID := parsing.ExtractTweetID(args[0])
			if tweetID == "" {
				return fmt.Errorf("invalid tweet ID or URL: %q", args[0])
			}

			c, err := resolveClient()
			if err != nil {
				return fmt.Errorf("auth: %w", err)
			}

			filterMode := "author_chain"
			switch filter {
			case "full":
				filterMode = "full_chain"
			case "author":
				filterMode = "author_chain"
			}

			opts := &types.ThreadOptions{
				FetchOptions: types.FetchOptions{
					Limit:      globalFlags.limit,
					MaxPages:   globalFlags.maxPages,
					QuoteDepth: resolveQuoteDepthFromCommand(),
					IncludeRaw: globalFlags.jsonFull,
				},
				FilterMode: filterMode,
			}

			tweets, err := c.GetThread(cmd.Context(), tweetID, opts)
			if err != nil {
				return fmt.Errorf("thread: %w", err)
			}

			if globalFlags.jsonOutput || globalFlags.jsonFull {
				return output.PrintJSON(cmd.OutOrStdout(), tweets)
			}
			for i := range tweets {
				printTweet(cmd, &tweets[i].TweetData)
				cmd.Println("---")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&filter, "filter", "", "thread filter mode: author|full")
	return cmd
}

func printTweet(cmd *cobra.Command, t *types.TweetData) {
	if t == nil {
		return
	}
	cmd.Printf("@%s (%s) [%s]\n%s\n", t.Author.Username, t.Author.Name, t.CreatedAt, t.Text)
	cmd.Printf("replies:%d retweets:%d likes:%d\n", t.ReplyCount, t.RetweetCount, t.LikeCount)
}
