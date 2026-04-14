package agent

import (
	"fmt"

	"github.com/spf13/cobra"
	"spwn.sh/packages/mind"
)

func init() {
	Cmd.AddCommand(dreamCmd)
}

var dreamCmd = &cobra.Command{
	Use:     "dream <agent-name>",
	Aliases: []string{"reflect"},
	Short:   "Analyze experience, discover patterns, promote playbooks",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s := newStepper(cmd)

		s.Blank()
		s.Start(fmt.Sprintf("Dreaming for agent %q...", name))

		result, err := mind.Dream(name)
		if err != nil {
			return s.FailHint("Dream failed", err,
				fmt.Sprintf("Check that agent %q exists with \"spwn agent inspect %s\"", name, name))
		}

		if result.Skipped {
			s.Info("Skipped", result.Reason)
			s.Info("Hint", "Journal entries are created when worlds are destroyed. Use \"spwn down <world>\" first.")
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
