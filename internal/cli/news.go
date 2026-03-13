package cli

import (
	"fmt"
	"strings"

	"github.com/mudrii/gobird/internal/client"
	"github.com/mudrii/gobird/internal/output"
	"github.com/mudrii/gobird/internal/types"
	"github.com/spf13/cobra"
)

func newNewsCmd() *cobra.Command {
	var tabsFlag string
	var limit int
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "news",
		Short: "Fetch news from explore tabs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			opts := buildNewsOpts(tabsFlag, limit, client.DefaultNewsTabs, globalFlags.jsonFull)
			items, err := c.GetNews(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if asJSON || globalFlags.jsonFull {
				return output.PrintJSON(cmd.OutOrStdout(), items)
			}
			fmtOpts := currentFormatOptions()
			for _, n := range items {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), output.FormatNewsItem(n, fmtOpts)); err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&tabsFlag, "tabs", "", "Comma-separated tab names (e.g. forYou,news)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of items")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")

	return cmd
}

func newTrendingCmd() *cobra.Command {
	var limit int
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "trending",
		Short: "Fetch trending topics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := quickClient()
			if err != nil {
				return err
			}
			opts := buildNewsOpts("trending", limit, nil, globalFlags.jsonFull)
			items, err := c.GetNews(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if asJSON || globalFlags.jsonFull {
				return output.PrintJSON(cmd.OutOrStdout(), items)
			}
			fmtOpts := currentFormatOptions()
			for _, n := range items {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), output.FormatNewsItem(n, fmtOpts)); err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of items")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output as JSON")

	return cmd
}

func buildNewsOpts(tabsFlag string, limit int, defaultTabs []string, includeRaw bool) *types.NewsOptions {
	var tabs []string
	if tabsFlag != "" {
		for _, t := range strings.Split(tabsFlag, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tabs = append(tabs, t)
			}
		}
	}
	if len(tabs) == 0 {
		tabs = defaultTabs
	}
	maxCount := limit
	if maxCount == 0 {
		maxCount = 20
	}
	return &types.NewsOptions{
		Tabs:       tabs,
		MaxCount:   maxCount,
		IncludeRaw: includeRaw,
	}
}
