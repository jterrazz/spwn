package agent

import (
	"fmt"
	"strings"

	agentDomain "spwn.sh/core/agent"
	"spwn.sh/core/foundation"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:     "new [name]",
	Aliases: []string{"init"},
	Short:   "Create a new agent with a 6-layer Mind",
	Long: `Create a new agent with the 6-layer Mind structure. If no name is
provided, a random name is picked from a curated dictionary.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newStepper(cmd)

		name := ""
		if len(args) > 0 {
			name = args[0]
		} else {
			name = foundation.RandomAgentName()
		}

		s.Blank()
		s.Start(fmt.Sprintf("Creating agent %q...", name))
		_, err := agentDomain.InitMind(name)
		if err != nil {
			hint := "Check that ~/.spwn/agents/ is writable"
			if strings.Contains(err.Error(), "already exists") {
				hint = fmt.Sprintf("Run \"spwn agent rm %s\" first, or choose a different name", name)
			}
			return s.FailHint("Agent creation failed", err, hint)
		}

		s.Done("Created agent", name)
		s.Done("Created persona", "default.md")

		s.Blank()
		s.Success(fmt.Sprintf("Spawn with: spwn up --agent %s", name))
		s.Blank()

		return nil
	},
}
