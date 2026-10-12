package cli

import (
	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/dependency"
)

func installCmd() *cobra.Command {
	var agentFilter string
	cmd := &cobra.Command{
		Use:   "install <ref>",
		Short: "Install a dependency into the project",
		Long: `Add a catalog, GitHub, or local dependency to agent manifests and pin it in spwn.lock.

Bare names resolve to the spwn: catalog ("spwn install qmd" installs spwn:qmd).
Without --agent, the ref is added to every agent in the project.
Local refs (skill/, tool/, hook/) require --agent to pick a target.

Examples:
  spwn install python                         # catalog dep, every agent
  spwn install spwn:python                    # explicit form, every agent
  spwn install github:jterrazz/research-skills
  spwn install qmd --agent mark               # catalog dep, only mark
  spwn install skill/refine --agent dylan     # local skill, only dylan`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dependency.RunInstall(cmd, args[0], agentFilter)
		},
	}
	cmd.Flags().StringVar(&agentFilter, "agent", "", "Target a single agent instead of every agent in the project")
	return cmd
}

func uninstallCmd() *cobra.Command {
	var agentFilter string
	cmd := &cobra.Command{
		Use:     "uninstall <ref>",
		Aliases: []string{"remove"},
		Short:   "Remove a dependency from the project",
		Long: `Remove a dependency from agent manifests. When no agent still carries the
ref, the lockfile pin is dropped too.

Without --agent, the ref is removed from every agent. Pass --agent <name>
to detach it from a single agent while leaving others untouched.

Examples:
  spwn uninstall python                     # every agent
  spwn uninstall skill/refine --agent mark  # only mark`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dependency.RunUninstall(cmd, args[0], agentFilter)
		},
	}
	cmd.Flags().StringVar(&agentFilter, "agent", "", "Target a single agent instead of every agent in the project")
	return cmd
}
