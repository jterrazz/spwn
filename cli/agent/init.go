package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/mind"
)

var initAgentCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Scaffold a new Mind",
	Args:  cobra.ExactArgs(1),
	RunE:  runInitAgent,
}

func runInitAgent(cmd *cobra.Command, args []string) error {
	name := args[0]

	fmt.Println()
	fmt.Printf("  Creating agent %q...\n", name)
	fmt.Println()

	dir, err := mind.Init(name)
	if err != nil {
		return err
	}

	fmt.Printf("  ✓ Created %s\n", dir)
	for _, layer := range config.MindLayers {
		label := "(empty)"
		if layer == "personas" {
			label = "(default.md)"
		}
		fmt.Printf("  ✓ Created %s/ %s\n", layer, label)
	}
	fmt.Println()
	fmt.Println("  Agent initialized. Edit personas/default.md to define identity.")
	fmt.Println()

	return nil
}
