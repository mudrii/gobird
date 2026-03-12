package cli

import (
	"context"
	"fmt"
	"os"

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
				fmt.Fprintln(os.Stderr, "FAIL:", err)
				os.Exit(1)
			}
			ctx := context.Background()
			u, err := c.GetCurrentUser(ctx)
			if err != nil {
				fmt.Fprintln(os.Stderr, "FAIL:", err)
				os.Exit(1)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "OK: @%s\n", u.Username)
			return nil
		},
	}
}
