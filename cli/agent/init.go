package agent

import (
	"fmt"

	"github.com/jterrazz/universe/internal/mind"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Create a new agent config template",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

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
		fmt.Printf("  Agent config initialized at %s/\n", path)
		fmt.Printf("  Edit personas/default.md to define identity.\n")
		fmt.Printf("  Spawn with: universe spawn --agent %s\n\n", name)

		return nil
	},
}
