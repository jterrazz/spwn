package cli

import (
	"fmt"
	"os"

	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/manifest"
	"github.com/jterrazz/universe/internal/wordlist"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "First-time setup — create ~/.universe/ and a named universe config",
	Long:  "Creates the ~/.universe/ directory structure and a named universe config. If no name is provided, a random cosmos-themed word is picked.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		} else {
			name = wordlist.PickConfig()
		}

		// Create base directory
		baseDir := config.BaseDir()
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			return fmt.Errorf("error: cannot create %s.\n%w", baseDir, err)
		}

		if !quiet {
			fmt.Println("\n  Initializing Universe...")
			fmt.Println()
		}

		// Create named universe config
		if err := manifest.CreateConfig(name); err != nil {
			return fmt.Errorf("error: cannot create config %q.\n%w", name, err)
		}

		if !quiet {
			fmt.Printf("  ✓ Created %s/universes/%s.yaml\n", baseDir, name)
			fmt.Println()
			fmt.Printf("  Ready. Create an agent with 'universe agent init' then spawn a world.\n")
			fmt.Printf("  Example: universe spawn %s --agent <agent-name> -w ./my-project\n\n", name)
		}

		return nil
	},
}
