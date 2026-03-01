package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(destroyCmd)
}

var destroyCmd = &cobra.Command{
	Use:   "destroy <universe-id>",
	Short: "Destroy a universe",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		universeID := args[0]

		arc, err := newArchitect()
		if err != nil {
			return err
		}

		u, err := arc.Destroy(ctx, universeID)
		if err != nil {
			return fmt.Errorf("error: cannot destroy universe %s.\n%w", universeID, err)
		}

		if !quiet {
			fmt.Printf("\n  Destroying universe %s...\n\n", universeID)
			fmt.Printf("  ✓ Stopped agent\n")
			fmt.Printf("  ✓ Removed container\n")
			if u.MindPath != "" {
				fmt.Printf("  ✓ Agent persisted at %s\n", u.MindPath)
			}
			fmt.Println()
			fmt.Println("  Universe destroyed. Agent survives.")
			fmt.Println()
		}

		return nil
	},
}
