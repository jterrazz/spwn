package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <id>",
	Short: "Open interactive session into a running universe",
	Args:  cobra.ExactArgs(1),
	RunE:  runAttach,
}

func init() {
	rootCmd.AddCommand(attachCmd)
}

func runAttach(cmd *cobra.Command, args []string) error {
	universeID := args[0]

	arch, err := newArchitect()
	if err != nil {
		return err
	}

	u, err := arch.Inspect(cmd.Context(), universeID)
	if err != nil {
		return fmt.Errorf("error: universe %s not found.\nRun 'universe list' to see available universes", universeID)
	}

	if !quiet {
		fmt.Println()
		fmt.Printf("  Attaching to universe %s...\n", universeID)
		agent := "—"
		if u.Agent != "" {
			agent = u.Agent
		}
		fmt.Printf("  Agent: %s | Origin: %s | Backend: %s\n", agent, u.Origin, u.Backend)
		fmt.Println()
	}

	_, err = arch.Attach(cmd.Context(), universeID)
	return err
}
