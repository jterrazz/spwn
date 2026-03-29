package agent

import (
	"fmt"

	agentDomain "github.com/jterrazz/spwn/core/agent"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(talkCmd)
}

var talkCmd = &cobra.Command{
	Use:   "talk <agent-name> <message>",
	Short: "Open a one-shot conversation with a named agent",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		message := args[1]
		s := newStepper(cmd)

		// Validate the agent exists
		if err := agentDomain.ValidateMind(name); err != nil {
			return fmt.Errorf("error: agent %q not found.\nRun 'spwn agent list' to see available agents.", name)
		}

		info, err := agentDomain.InspectAgent(name)
		if err != nil {
			return fmt.Errorf("error: cannot inspect agent %q.\n%w", name, err)
		}

		s.Blank()
		s.Info("Agent:", info.Name)
		s.Info("Mind:", info.Path)
		s.Blank()

		// Show recent journal entries
		entries, err := agentDomain.ListJournal(info.Path, 5)
		if err == nil && len(entries) > 0 {
			s.Info("Recent journal:", "")
			for _, e := range entries {
				ts := e.CreatedAt.Format("2006-01-02 15:04")
				s.Info("  "+ts, fmt.Sprintf("%-24s %s", e.UniverseID, e.Outcome))
			}
			s.Blank()
		}

		s.Info("Message:", message)
		s.Blank()
		s.Log("Direct agent conversation not yet fully implemented — requires active universe.")
		s.Blank()

		return nil
	},
}
