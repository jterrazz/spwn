package cli

import (
	"spwn.sh/apps/cli/agent"
	"spwn.sh/apps/cli/claw"
	"spwn.sh/apps/cli/observatory"
	"spwn.sh/apps/cli/skill"
	"spwn.sh/apps/cli/world"
	"spwn.sh/apps/cli/visitor"
	"github.com/spf13/cobra"
)

// Version is set by goreleaser via ldflags.
var Version = "dev"

var (
	jsonOutput bool
	quiet      bool
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "spwn",
	Short: "spwn — create realities for things that can think",
	Long: `spwn creates isolated Docker environments for AI agents.
Each world has physics (constants, laws, elements),
and a Mind (persistent agent identity).`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return ensureDefaults()
	},
}

func init() {
	rootCmd.Version = Version

	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show debug information")

	rootCmd.AddCommand(world.Cmd)
	rootCmd.AddCommand(agent.Cmd)
	rootCmd.AddCommand(claw.Cmd)
	rootCmd.AddCommand(visitor.Cmd)
	rootCmd.AddCommand(observatory.Cmd)
	rootCmd.AddCommand(skill.Cmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// GetRootCmd returns the root command for documentation generation.
func GetRootCmd() *cobra.Command {
	return rootCmd
}
