package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(attachCmd)
}

var attachCmd = &cobra.Command{
	Use:   "attach <universe-id>",
	Short: "Open interactive session into a running universe",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		universeID := args[0]

		arc, err := newArchitect()
		if err != nil {
			return err
		}

		if !quiet {
			fmt.Printf("\n  Attaching to universe %s...\n\n", universeID)
		}

		return arc.Attach(ctx, universeID)
	},
}
