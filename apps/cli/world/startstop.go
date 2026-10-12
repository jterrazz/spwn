package world

import (
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:     "start [name]",
	Aliases: []string{"up"},
	Short:   "Start a world (alias for `spwn up`)",
	Args:    cobra.MaximumNArgs(1),
	RunE:    composeUpRunE,
}

var stopCmd = &cobra.Command{
	Use:     "stop [name-or-id]",
	Aliases: []string{"down"},
	Short:   "Stop a world (alias for `spwn down`)",
	Args:    cobra.MaximumNArgs(1),
	RunE:    composeDownRunE,
}

func init() {
	registerSpawnFlags(startCmd)
	stopCmd.Flags().BoolVar(&destroyAll, "all", false, "Destroy all running worlds")
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(stopCmd)
}
