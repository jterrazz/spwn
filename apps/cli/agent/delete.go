package agent

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/agent"
	"spwn.sh/packages/project"
)

func init() {
	Cmd.AddCommand(deleteCmd)
}

var deleteCmd = &cobra.Command{
	Use:     "rm <agent-name>",
	Aliases: []string{"delete"},
	Short:   "Remove an agent and its Mind directory",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s := ui.New()

		s.Blank()
		s.Start(fmt.Sprintf("Deleting agent %q...", name))

		if err := agent.DeleteAgent(name); err != nil {
			return s.FailHint("Delete failed", err,
				fmt.Sprintf("Check that agent %q exists with \"spwn agent ls\"", name))
		}

		s.Done("Deleted agent", name)

		// Keep spwn.yaml in sync when inside a project: scrub any
		// world references to the deleted agent (symmetric with the
		// auto-world that `agent create` adds). Without this, the
		// next `spwn check` fails with "agent directory not found".
		if cwd, werr := os.Getwd(); werr == nil {
			if p, perr := project.Find(cwd); perr == nil && p != nil {
				if err := project.RemoveAgentFromManifest(p.ManifestPath, name); err != nil {
					s.Warn("Warning", fmt.Sprintf("could not update spwn.yaml: %v", err))
				} else {
					s.Done("Updated spwn.yaml", name)
				}
			}
		}

		s.Blank()

		return nil
	},
}
