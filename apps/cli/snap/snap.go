package snap

import (
	"spwn.sh/apps/cli/ui"
	"github.com/spf13/cobra"
)

var defaultSnapHelp func(*cobra.Command, []string)

func init() {
	defaultSnapHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(snapHelp)
}

func snapHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "snap" {
		if defaultSnapHelp != nil {
			defaultSnapHelp(cmd, args)
		}
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ world snap")+" "+ui.Faint("- world snapshots"),
		[]ui.HelpGroup{
			{Title: "Commands", Commands: []ui.HelpEntry{
				{Name: "save <world-id>", Desc: "Save world state"},
				{Name: "ls", Desc: "List all snapshots"},
				{Name: "restore <snap-id>", Desc: "Restore from snapshot"},
				{Name: "rm <snap-id>", Desc: "Remove a snapshot"},
			}},
		},
		"spwn world snap [command]",
		"Use \"spwn world snap <command> --help\" for more information.",
	)
}

// Cmd is the snap command group, attached to `spwn world` at CLI
// registration time in apps/cli/root.go.
var Cmd = &cobra.Command{
	Use:   "snap",
	Short: "World snapshots - save, ls, restore, rm",
	Long:  `Save, list, restore, and remove world snapshots.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
