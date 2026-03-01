package cli

import (
	"fmt"
	"os"

	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/manifest"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "First-time setup — create ~/.universe/ and a named universe config",
	Long: `First-time setup. Creates the ~/.universe/ directory structure and a named
universe config. If no name is provided, a random cosmos-themed word is picked.

On first run, also creates default.yaml as the default config.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		} else {
			name = config.RandomCosmosWord()
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

		// Create default universe config
		if err := manifest.CreateDefault(); err != nil {
			if !quiet {
				fmt.Printf("  (default.yaml already exists, skipping)\n")
			}
		} else if !quiet {
			fmt.Printf("  ✓ Created %s/universes/default.yaml\n", baseDir)
		}

		// Create named config
		if name != "default" {
			if err := manifest.CreateConfig(name); err != nil {
				if !quiet {
					fmt.Printf("  (%s.yaml already exists, skipping)\n", name)
				}
			} else if !quiet {
				fmt.Printf("  ✓ Created %s/universes/%s.yaml\n", baseDir, name)
			}
		}

		if !quiet {
			fmt.Println()
			fmt.Println("  Ready. Next steps:")
			fmt.Printf("    universe agent init              # create an agent\n")
			fmt.Printf("    universe spawn --agent <name>    # create your first world\n")
			fmt.Println()
		}

		return nil
	},
}
