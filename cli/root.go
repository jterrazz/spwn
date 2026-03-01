package cli

import (
	"github.com/spf13/cobra"

	"github.com/jterrazz/universe/cli/agent"
)

var (
	jsonOutput bool
	quiet      bool
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "universe",
	Short: "Universe — create realities for things that can think.",
	Long: `Universe — create realities for things that can think.

Create isolated, sandboxed worlds where AI agents can operate safely.
Each universe gets its own physics (resource limits, network rules, available tools)
and can host a single agent backed by a persistent Mind.`,
	SilenceUsage:  true,
	SilenceErrors: true,
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
