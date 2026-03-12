package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mudrii/gobird/internal/client"
	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/types"
	"github.com/spf13/cobra"
)

func quickClient() (*client.Client, error) {
	authToken := os.Getenv("AUTH_TOKEN")
	ct0 := os.Getenv("CT0")
	if authToken == "" || ct0 == "" {
		return nil, fmt.Errorf("AUTH_TOKEN and CT0 must be set")
	}
	return client.New(authToken, ct0, nil), nil
}

func newHomeCmd() *cobra.Command {
	var limit int
	var asJSON bool
	var latest bool

	cmd := &cobra.Command{
		Use:   "home",
		Short: "Fetch home timeline",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			ctx := context.Background()
			opts := &types.FetchOptions{Limit: limit}
			var result types.TweetResult
			if latest {
				result = c.GetHomeLatestTimeline(ctx, opts)
			} else {
				result = c.GetHomeTimeline(ctx, opts)
			}
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
	cmd.Flags().BoolVar(&latest, "latest", false, "Use latest (chronological) timeline")

	return cmd
}

// stripAtPrefix removes a leading '@' from a handle if present.
func stripAtPrefix(handle string) string {
	return strings.TrimPrefix(handle, "@")
}
