package cli

import (
	"fmt"
	"os"

	"github.com/jterrazz/spwn/apps/cli/ui"
	"github.com/jterrazz/spwn/core/universe"
	"github.com/jterrazz/spwn/core/foundation"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "First-time setup — create ~/.spwn/ and a named universe config",
	Long: `First-time setup. Creates the ~/.spwn/ directory structure and a named
universe config. If no name is provided, a random cosmos-themed word is picked.

On first run, also creates default.yaml as the default config.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := ui.New(quiet, verbose, jsonOutput)

		name := ""
		if len(args) > 0 {
			name = args[0]
		} else {
			name = foundation.RandomCosmosWord()
		}

		// Create base directory
		baseDir := foundation.BaseDir()
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			return fmt.Errorf("error: cannot create %s.\n%w", baseDir, err)
		}

		s.Blank()

		// Create organization manifest
		if err := universe.CreateOrg(name); err != nil {
			s.Log("org.yaml already exists, skipping")
		} else {
			s.Done("Created organization", "org.yaml")
		}

		// Create default universe config
		if err := universe.CreateDefaultConfig(); err != nil {
			s.Log("default.yaml already exists, skipping")
		} else {
			s.Done("Created config", "default.yaml")
		}

		// Create named config
		if name != "default" {
			if err := universe.CreateConfig(name); err != nil {
				s.Log("%s.yaml already exists, skipping", name)
			} else {
				s.Done("Created config", name+".yaml")
			}
		}

		s.Blank()
		s.Success(fmt.Sprintf("Ready. Run: spwn universe -c %s", name))
		s.Blank()

		return nil
	},
}
