package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/agent"
	"spwn.sh/packages/project"
)

func init() {
	Cmd.AddCommand(forkCmd)
	ui.MarkExperimental(forkCmd)
}

var forkCmd = &cobra.Command{
	Use:   "fork <source> <target>",
	Short: "Clone a Mind from one agent to another",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]
		target := args[1]
		s := newStepper(cmd)

		s.Blank()
		s.Start(fmt.Sprintf("Forking %q -> %q...", source, target))

		result, err := agent.Fork(source, target, nil)
		if err != nil {
			return s.FailHint("Fork failed", err, "Check agents exist with \"spwn agent ls\"")
		}

		s.Done("Source", result.Source)
		s.Done("Target", result.Target)
		s.Done("Layers copied", strings.Join(result.LayersCopied, ", "))

		// Auto-world for the forked agent, symmetric with
		// `agent create`. Without this, `spwn check` flags the
		// forked agent as an orphan — surprising and silent.
		if cwd, werr := os.Getwd(); werr == nil {
			if p, perr := project.Find(cwd); perr == nil && p != nil {
				if err := project.AddAgentToManifest(p.ManifestPath, target); err != nil {
					s.Warn("Warning", fmt.Sprintf("could not add world to spwn.yaml: %v", err))
				} else {
					s.Done("Added world to spwn.yaml", target)
				}
			}
		}

		s.Blank()

		return nil
	},
}
