package agent

import (
	"fmt"

	"spwn.sh/apps/cli/ui"
	agentDomain "spwn.sh/core/agent"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(sessionsCmd)
}

var sessionsCmd = &cobra.Command{
	Use:   "sessions <agent-name>",
	Short: "View an agent's session history",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s := newStepper(cmd)

		mindPath := agentDomain.AgentDir(name)
		if err := agentDomain.ValidateMind(name); err != nil {
			return fmt.Errorf("agent %q not found", name)
		}

		sessions, err := agentDomain.ListSessions(mindPath)
		if err != nil {
			return fmt.Errorf("cannot read sessions: %w", err)
		}

		if len(sessions) == 0 {
			s.Blank()
			s.Success("No sessions.")
			s.Log("Spawn the agent into a world to create sessions.")
			s.Blank()
			return nil
		}

		t := ui.NewTable(ui.ModeNormal, "WORLD", "SESSION ID", "RESUMED")
		for _, sess := range sessions {
			resumed := "no"
			if sess.Resumed {
				resumed = "yes"
			}
			t.AddRow(sess.WorldID, sess.ID, resumed)
		}
		t.Render()

		return nil
	},
}
