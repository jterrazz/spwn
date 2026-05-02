// Package automation implements the `spwn automation` command group.
//
// Subcommands:
//
//	ls      — list every automation declared in spwn.yaml plus its
//	          last-fired status (cron) or watcher path (fs).
//	logs    — tail the project's <root>/.spwn/runs.jsonl receipt log,
//	          either as a one-shot dump or with --follow.
//	status  — engine state snapshot: how many automations are
//	          registered per world, last fire times, freshness.
//	daemon  — run the engine for the current project and block until
//	          interrupted. The architect daemon (long-running) is the
//	          eventual home; for now this is the simplest "run my
//	          automations" path.
//
// All four are read-only against the project filesystem except
// `daemon`, which writes receipts + state.
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
to one of your project's agents. The architect daemon fires them as
events arrive and writes a receipt for every dispatch.

Declare automations under spwn.yaml#worlds.<name>.automations. See
docs/automations.md for the full schema.`,
}

func init() {
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(logsCmd)
	Cmd.AddCommand(statusCmd)
	Cmd.AddCommand(daemonCmd)

	Cmd.SetHelpFunc(automationHelp)
}

// automationHelp keeps the help screen consistent with the rest of
// the CLI (skill, agent, world all use ui.RenderGroupedHelp).
func automationHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "automation" {
		ui.MinimalHelp(cmd, args)
		return
	}
	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ automation")+" "+ui.Faint("- trigger-driven agent wakeups"),
		[]ui.HelpGroup{
			{Title: "Inspect", Commands: []ui.HelpEntry{
				{Name: "ls", Desc: "List every declared automation"},
				{Name: "status", Desc: "Engine state and last-fired snapshot"},
				{Name: "logs", Desc: "Tail .spwn/runs.jsonl receipts"},
			}},
			{Title: "Run", Commands: []ui.HelpEntry{
				{Name: "daemon", Desc: "Run the engine until interrupted"},
			}},
		},
		"spwn automation [command]",
		"",
	)
}
