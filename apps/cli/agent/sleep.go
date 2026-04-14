package agent

import (
	"fmt"

	agentDomain "spwn.sh/packages/agent"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(sleepCmd)
}

var sleepCmd = &cobra.Command{
	Use:   "sleep <agent-name>",
	Short: "Consolidate experience - archive stale files, prune old sessions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s := newStepper(cmd)

		s.Blank()
		s.Start(fmt.Sprintf("Sleep cycle for agent %q...", name))

		result, err := agentDomain.Sleep(name)
		if err != nil {
			return s.FailHint("Sleep failed", err,
				fmt.Sprintf("Check that agent %q exists with \"spwn agent inspect %s\"", name, name))
		}

		s.Done("Archived playbooks", fmt.Sprintf("%d", result.ArchivedPlaybooks))
		s.Done("Archived knowledge", fmt.Sprintf("%d", result.ArchivedKnowledge))
		s.Done("Pruned sessions", fmt.Sprintf("%d", result.PrunedSessions))
		s.Blank()

		return nil
	},
}
