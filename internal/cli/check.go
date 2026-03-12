package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Validate credentials and print current user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := quickClient()
			if err != nil {
				return fmt.Errorf("FAIL: %w", err)
			}
			u, err := c.GetCurrentUser(cmd.Context())
			if err != nil {
				return fmt.Errorf("FAIL: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "OK: @%s\n", u.Username)
			return nil
		},
	}
}
