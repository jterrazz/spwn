package visitor

import (
	"fmt"

	"github.com/spf13/cobra"
)

var universeFlag string

// Cmd spawns an ephemeral agent with a single task.
var Cmd = &cobra.Command{
	Use:   "visitor [task]",
	Short: "Spawn an ephemeral agent — no Mind, no memory, fire and forget",
	Long: `Visitors are ephemeral agents that execute a single task and disappear.
They have no persistent identity, no Mind, and no journal.
Perfect for linting, testing, health checks, and one-off validations.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		task := args[0]
		if universeFlag == "" {
			return fmt.Errorf("error: --universe is required.\nRun 'spwn universe list' to see active universes.")
		}
		fmt.Printf("  Visitor dispatched: %q → %s\n", task, universeFlag)
		return nil
	},
}

func init() {
	Cmd.Flags().StringVar(&universeFlag, "universe", "", "Target universe ID (required)")
	Cmd.MarkFlagRequired("universe")
}
