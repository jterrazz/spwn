package architect

import (
	"spwn.sh/apps/cli/logs"
	"github.com/spf13/cobra"
)

var architectLogsLimit int

func init() {
	logsCmd.Flags().IntVarP(&architectLogsLimit, "limit", "n", 50, "Number of events to show")
	Cmd.AddCommand(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show the Architect daemon's event log",
	RunE: func(cmd *cobra.Command, args []string) error {
		return logs.Run(cmd, logs.RunOpts{
			Limit: architectLogsLimit,
			Actor: "architect",
		})
	},
}
