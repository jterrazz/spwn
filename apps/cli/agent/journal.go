package agent

import (
	"fmt"

	"spwn.sh/apps/cli/ui"
	agentDomain "spwn.sh/core/agent"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(journalCmd)
}

var journalCmd = &cobra.Command{
	Use:   "journal <agent-name>",
	Short: "View an agent's journal history",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s := newStepper(cmd)

		mindPath := agentDomain.AgentDir(name)
		if err := agentDomain.ValidateMind(name); err != nil {
			return fmt.Errorf("error: agent %q not found.\nRun 'spwn agent list' to see available agents.", name)
		}

		entries, err := agentDomain.ListJournal(mindPath, 20)
		if err != nil {
			return fmt.Errorf("error: cannot read journal.\n%w", err)
		}

		if len(entries) == 0 {
			s.Blank()
			s.Success("No journal entries.")
			s.Log("Spawn the agent into a world to create journal entries.")
			s.Blank()
			return nil
		}

		t := ui.NewTable(ui.ModeNormal, "DATE", "WORLD", "EXIT", "DURATION")
		for _, e := range entries {
			t.AddRow(
				e.CreatedAt.Format("2006-01-02"),
				e.UniverseID,
				fmt.Sprintf("%d", e.ExitCode),
				ui.FormatDuration(e.Duration),
			)
		}
		t.Render()

		return nil
	},
}
