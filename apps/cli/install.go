package cli

import (
	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/pack"
)

// installCmd creates the root-level `spwn install <ref>` command.
// Delegates to pack.RunInstall — same logic as `spwn pack install`,
// just at the top level for convenience (like `go get`, `npm install`).
func installCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <ref>",
		Short: "Install a dependency into the project",
		Long: `Add a catalog or GitHub pack to every agent's deps and pin it in spwn.lock.

Examples:
  spwn install @spwn/python
  spwn install github.com/jterrazz/research-skills`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pack.RunInstall(cmd, args[0])
		},
	}
}

// uninstallCmd creates the root-level `spwn uninstall <ref>` command.
func uninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "uninstall <ref>",
		Aliases: []string{"remove"},
		Short:   "Remove a dependency from the project",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pack.RunUninstall(cmd, args[0])
		},
	}
}
