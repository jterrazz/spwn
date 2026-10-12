package world

import (
	"github.com/spf13/cobra"
	"spwn.sh/apps/cli/logs"
)

var logsLimit int

func init() {
	logsCmd.Flags().IntVarP(&logsLimit, "limit", "n", 50, "Number of events to show")
	Cmd.AddCommand(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs <world-id>",
	Short: "Show the event log for a specific world",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return logs.Run(cmd, logs.RunOpts{
			Limit:               logsLimit,
			WorldID:             args[0],
			SkipWorldValidation: true,
		})
	},
}
