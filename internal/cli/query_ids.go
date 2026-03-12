package cli

import (
	"github.com/mudrii/gobird/internal/client"
	"github.com/mudrii/gobird/internal/output"
	"github.com/spf13/cobra"
)

func newQueryIDsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "query-ids",
		Short: "Print the current fallback query ID cache",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return output.PrintJSON(cmd.OutOrStdout(), client.FallbackQueryIDs)
		},
	}
}
