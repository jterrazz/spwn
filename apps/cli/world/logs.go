package world

import (
	"spwn.sh/apps/cli/logs"
	"github.com/spf13/cobra"
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
			Limit: logsLimit,
			// The scoped `world logs <id>` shortcut accepts a live
			// world ID (e.g. `w-abc123`), which never appears in
			// spwn.yaml — skip the manifest-backed filter check so
			// the argument isn't rejected as "unknown world".
			WorldID:             args[0],
			SkipWorldValidation: true,
		})
	},
}
