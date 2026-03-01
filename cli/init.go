package cli

import (
	"fmt"
	"os"

	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/manifest"
	"github.com/jterrazz/universe/internal/mind"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "First-time setup — create ~/.universe/ with a named config and agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Create base directory
		baseDir := config.BaseDir()
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			return fmt.Errorf("error: cannot create %s.\n%w", baseDir, err)
		}

		if !quiet {
			fmt.Println("\n  Initializing Universe...")
			fmt.Println()
		}

		// Create default universe config
		if err := manifest.CreateDefault(); err != nil {
			if !quiet {
				fmt.Printf("  (default.yaml already exists, skipping)\n")
			}
		} else if !quiet {
			fmt.Printf("  ✓ Created %s/universes/default.yaml\n", baseDir)
		}

		// Create named agent
		path, err := mind.Init(name)
		if err != nil {
			return fmt.Errorf("error: cannot create agent %q.\n%w", name, err)
		}

		if !quiet {
			fmt.Printf("  ✓ Created agent %q (6-layer Mind)\n", name)
			fmt.Println()
			fmt.Printf("  Ready. Run 'universe spawn --agent %s' to create your first world.\n", name)
			fmt.Printf("  Run 'universe config init node-dev' to create a named universe config.\n")
			_ = path
		}

		return nil
	},
}
