// Package automation implements the `spwn automation` command group.
//
// Its one subcommand is daemon: run the engine for the current project
// and block until interrupted, writing receipts and state. Reading the
// automations and their runs is `station automations`.
package automation

import (
	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
)

// Cmd is the root `spwn automation` command group.
var Cmd = &cobra.Command{
	Use:   "automation",
	Short: "Trigger-driven agent wakeups (cron + filesystem)",
	Long: `Automations bind a trigger (cron expression or filesystem watch)
to one of your project's agents. The command "spwn automation daemon" fires them as
events arrive and writes a receipt for every dispatch.

Declare automations under spwn.yaml#worlds.<name>.automations. See
docs/guides/automations.md for the full schema. "station automations" lists
them and the runs written for them.`,
}

func init() {
	Cmd.AddCommand(daemonCmd)

	Cmd.SetHelpFunc(automationHelp)
}

func automationHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "automation" {
		ui.MinimalHelp(cmd, args)
		return
	}
	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ automation")+" "+ui.Faint("- trigger-driven agent wakeups"),
		[]ui.HelpGroup{
			{Title: "Run", Commands: []ui.HelpEntry{
				{Name: "daemon", Desc: "Run the engine until interrupted"},
			}},
		},
		"spwn automation [command]",
		"",
	)
}
