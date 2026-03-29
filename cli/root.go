package cli

import (
	"github.com/jterrazz/spwn/cli/agent"
	"github.com/jterrazz/spwn/cli/universe"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	quiet      bool
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "spwn",
	Short: "spwn — create realities for things that can think",
	Long: `spwn creates isolated Docker environments for AI agents.
Each universe has physics (constants, laws, elements),
and a Mind (persistent agent identity).`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return ensureDefaults()
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show debug information")

	rootCmd.AddCommand(universe.Cmd)
	rootCmd.AddCommand(agent.Cmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
