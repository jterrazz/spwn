package agent

import (
	"fmt"
	"strings"

	agentDomain "github.com/jterrazz/spwn/core/agent"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(forkCmd)
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
			s.Fail("Fork failed", err)
			return fmt.Errorf("error: fork failed.\n%w", err)
		}

		s.Done("Source", result.Source)
		s.Done("Target", result.Target)
		s.Done("Layers copied", strings.Join(result.LayersCopied, ", "))
		s.Blank()

		return nil
	},
}
