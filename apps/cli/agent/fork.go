package agent

import (
	"fmt"
	"strings"

	"spwn.sh/apps/cli/ui"
	agentDomain "spwn.sh/packages/mind"
	"github.com/spf13/cobra"
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

		result, err := agentDomain.Fork(source, target, nil)
		if err != nil {
			return s.FailHint("Fork failed", err, "Check agents exist with \"spwn agent ls\"")
		}

		s.Done("Source", result.Source)
		s.Done("Target", result.Target)
		s.Done("Layers copied", strings.Join(result.LayersCopied, ", "))
		s.Blank()

		return nil
	},
}
