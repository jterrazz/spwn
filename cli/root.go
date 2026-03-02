package cli

import (
	"github.com/jterrazz/universe/cli/agent"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	quiet      bool
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "universe",
	Short: "Universe — create realities for things that can think",
	Long: `Universe creates isolated Docker environments for AI agents.
Each universe has physics (constants, laws, elements),
and a Mind (persistent agent identity).`,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show debug information")

	rootCmd.AddCommand(agent.Cmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
