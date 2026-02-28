package commands

import "github.com/spf13/cobra"

// Root returns the root cobra command.
func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "universe",
		Short: "Sandboxed AI agent environments",
	}

	cmd.AddCommand(createCmd())
	cmd.AddCommand(spawnCmd())
	cmd.AddCommand(listCmd())
	cmd.AddCommand(inspectCmd())
	cmd.AddCommand(destroyCmd())

	return cmd
}
