package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy <id>",
	Short: "Destroy a universe",
	Args:  cobra.ExactArgs(1),
	RunE:  runDestroy,
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}

func runDestroy(cmd *cobra.Command, args []string) error {
	universeID := args[0]

	arch, err := newArchitect()
	if err != nil {
		return err
	}

	// Get universe info before destroying (for output).
	u, err := arch.Inspect(cmd.Context(), universeID)
	if err != nil {
		return fmt.Errorf("error: universe %s not found.\nRun 'universe list' to see available universes", universeID)
	}

	if !quiet {
		fmt.Println()
		fmt.Printf("  Destroying universe %s...\n", universeID)
		fmt.Println()
	}

	if err := arch.Destroy(cmd.Context(), universeID); err != nil {
		return err
	}

	if !quiet {
		fmt.Println("  ✓ Stopped agent")
		fmt.Println("  ✓ Removed container")
		if u.MindPath != "" {
			fmt.Printf("  ✓ Mind persisted at %s\n", u.MindPath)
		}
		fmt.Println()
		fmt.Println("  Universe destroyed. Mind survives.")
		fmt.Println()
	}

	return nil
}
