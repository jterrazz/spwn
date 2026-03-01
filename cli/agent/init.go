package agent

import (
	"fmt"

	"github.com/jterrazz/universe/internal/mind"
	"github.com/jterrazz/universe/internal/wordlist"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Create a new agent with a 6-layer Mind",
	Long:  "Creates a new agent directory with the 6-layer Mind structure. If no name is provided, a random name is picked.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		} else {
			name = wordlist.PickAgent()
		}

		path, err := mind.Init(name)
		if err != nil {
			return fmt.Errorf("error: cannot create agent %q.\n%w", name, err)
		}

		fmt.Printf("\n  Creating agent %q...\n\n", name)
		fmt.Printf("  ✓ Created %s/\n", path)
		fmt.Printf("  ✓ Created personas/default.md\n")
		fmt.Printf("  ✓ Created skills/       (empty)\n")
		fmt.Printf("  ✓ Created knowledge/    (empty)\n")
		fmt.Printf("  ✓ Created playbooks/    (empty)\n")
		fmt.Printf("  ✓ Created journal/      (empty)\n")
		fmt.Printf("  ✓ Created sessions/     (empty)\n")
		fmt.Println()
		fmt.Printf("  Agent initialized at %s/\n", path)
		fmt.Printf("  Edit personas/default.md to define identity.\n")
		fmt.Printf("  Spawn with: universe spawn --agent %s\n\n", name)

		return nil
	},
}
