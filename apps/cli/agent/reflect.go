package agent

import (
	"fmt"

	agentDomain "github.com/jterrazz/spwn/core/agent"
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
			s.Fail("Reflexion failed", err)
			return fmt.Errorf("error: reflexion failed.\n%w", err)
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
