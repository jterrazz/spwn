package agent

import (
	"spwn.sh/apps/cli/logs"
	"github.com/spf13/cobra"
)

var agentLogsLimit int

func init() {
	agentLogsCmd.Flags().IntVarP(&agentLogsLimit, "limit", "n", 50, "Number of events to show")
	Cmd.AddCommand(agentLogsCmd)
}

var agentLogsCmd = &cobra.Command{
	Use:   "logs <agent-name>",
	Short: "Show the event log for a specific agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return logs.Run(cmd, logs.RunOpts{
			Limit:   agentLogsLimit,
			AgentID: args[0],
		})
	},
}
