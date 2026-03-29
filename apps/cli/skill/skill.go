package skill

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Cmd is the parent command for skill management.
var Cmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage skills — reusable capabilities for agents",
	Long:  `Skills are reusable capabilities that agents can use. Install from local files, git repos, or the marketplace.`,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("  No skills installed.")
		fmt.Println("  Run 'spwn skill install <source>' to add one.")
		return nil
	},
}

var installCmd = &cobra.Command{
	Use:   "install [source]",
	Short: "Install a skill from local path, git repo, or marketplace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("  Installing skill from %q...\n", args[0])
		fmt.Println("  (Not yet implemented — coming in Epoch 10)")
		return nil
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove an installed skill",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("  Removing skill %q...\n", args[0])
		return nil
	},
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(installCmd)
	Cmd.AddCommand(removeCmd)
}
