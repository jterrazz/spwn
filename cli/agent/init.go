package agent

import (
	"fmt"

	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/mind"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Create a new agent with a 6-layer Mind",
	Long: `Create a new agent with the 6-layer Mind structure. If no name is
provided, a random name is picked from a curated dictionary.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := newStepper(cmd)

		name := ""
		if len(args) > 0 {
			name = args[0]
		} else {
			name = config.RandomAgentName()
		}

		s.Blank()
		s.Start(fmt.Sprintf("Creating agent %q...", name))

		_, err := mind.Init(name)
		if err != nil {
			s.Fail("Agent creation failed", err)
			return fmt.Errorf("error: cannot create agent %q.\n%w", name, err)
		}

		s.Done("Created agent", name)
		s.Done("Created persona", "default.md")

		s.Blank()
		s.Success(fmt.Sprintf("Spawn with: universe spawn --agent %s", name))
		s.Blank()

		return nil
	},
}
