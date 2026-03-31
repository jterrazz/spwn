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
		ui.Strong("⬡ snap")+" "+ui.Faint("— world snapshots"),
		[]ui.HelpGroup{
			{Title: "Commands", Commands: []ui.HelpEntry{
				{"save <world-id>", "Save world state"},
				{"ls", "List all snapshots"},
				{"restore <snap-id>", "Restore from snapshot"},
				{"rm <snap-id>", "Remove a snapshot"},
			}},
		},
		"spwn snap [command]",
		"Use \"spwn snap <command> --help\" for more information.",
	)
}

// Cmd is the snap command group.
var Cmd = &cobra.Command{
	Use:   "snap",
	Short: "World snapshots — save, ls, restore, rm",
	Long:  `Save, list, restore, and remove world snapshots.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}
