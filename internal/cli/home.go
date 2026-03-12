package cli

import (
	"fmt"
	"strings"

	"github.com/mudrii/gobird/internal/client"
	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/types"
	"github.com/spf13/cobra"
)

func quickClient() (*client.Client, error) {
	return resolveClient()
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
			opts := &types.FetchOptions{Limit: limit, IncludeRaw: globalFlags.jsonFull, QuoteDepth: resolveQuoteDepthFromCommand()}
			var result types.TweetResult
			if latest {
				result = c.GetHomeLatestTimeline(cmd.Context(), opts)
			} else {
				result = c.GetHomeTimeline(cmd.Context(), opts)
			}
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
	cmd.Flags().BoolVar(&latest, "latest", false, "Use latest (chronological) timeline")

	return cmd
}

// stripAtPrefix removes a leading '@' from a handle if present.
func stripAtPrefix(handle string) string {
	return strings.TrimPrefix(handle, "@")
}
