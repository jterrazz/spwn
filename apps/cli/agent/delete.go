package agent

import (
	"fmt"

	agentDomain "spwn.sh/core/agent"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(deleteCmd)
}

var deleteCmd = &cobra.Command{
	Use:   "delete <agent-name>",
	Short: "Remove an agent and its Mind directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s := newStepper(cmd)

		s.Blank()
		s.Start(fmt.Sprintf("Deleting agent %q...", name))

		if err := agentDomain.DeleteAgent(name); err != nil {
			return s.FailHint("Delete failed", err,
				fmt.Sprintf("Check that agent %q exists with \"spwn agent list\"", name))
		}

		s.Done("Deleted agent", name)
		s.Blank()

		return nil
	},
}
