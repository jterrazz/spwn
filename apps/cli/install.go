package cli

import (
	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/dependency"
)

// installCmd creates the root-level `spwn install <ref>` command.
// Delegates to dependency.RunInstall for the actual install logic.
// Like `go get` or `npm install` — one verb, no subcommand.
func installCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <ref>",
		Short: "Install a dependency into the project",
		Long: `Add a catalog or GitHub dependency to every agent's manifest and pin it in spwn.lock.

Bare names resolve to the spwn: catalog ("spwn install qmd" installs spwn:qmd).

Examples:
  spwn install python
  spwn install spwn:python
  spwn install github:jterrazz/research-skills`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dependency.RunInstall(cmd, args[0])
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
			return dependency.RunUninstall(cmd, args[0])
		},
	}
}
