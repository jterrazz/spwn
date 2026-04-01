package agent

import (
	"fmt"

	agentDomain "spwn.sh/core/agent"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(reflectCmd)
}

var reflectCmd = &cobra.Command{
	Use:   "reflect <agent-name>",
	Short: "Analyze journal and promote patterns to playbooks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s := newStepper(cmd)

		s.Blank()
		s.Start(fmt.Sprintf("Reflecting on agent %q...", name))

		result, err := agentDomain.Reflect(name)
		if err != nil {
			return s.FailHint("Reflect failed", err,
				fmt.Sprintf("Check that agent %q exists with \"spwn agent inspect %s\"", name, name))
		}

		if result.Skipped {
			s.Info("Skipped", result.Reason)
			s.Blank()
			return nil
		}

		s.Done("Entries analyzed", fmt.Sprintf("%d", result.EntriesAnalyzed))
		s.Done("Completed", fmt.Sprintf("%d", result.CompletedTasks))
		s.Done("Failed", fmt.Sprintf("%d", result.FailedTasks))
		s.Done("Success rate", fmt.Sprintf("%.0f%%", result.SuccessRate*100))
		s.Done("Written to", result.OutputPath)
		s.Blank()

		return nil
	},
}
