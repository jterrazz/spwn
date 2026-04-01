package dash

import (
	"fmt"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

var defaultDashHelp func(*cobra.Command, []string)

// Cmd is the parent command for Dashboard operations.
var Cmd = &cobra.Command{
	Use:   "dash",
	Short: "Visual dashboard",
	Long:  `The dashboard — a real-time visual dashboard showing all worlds, agents, and their evolution.`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the dashboard server",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := universe.NewStore()
		if err != nil {
			return err
		}
		srv := universe.NewObservatoryServer(store, ":3001")
		fmt.Println("  Dashboard API on http://localhost:3001")
		return srv.Start()
	},
}

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open in browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("  (Not yet implemented — coming in Epoch 7)")
		return nil
	},
}

func init() {
	defaultDashHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(dashHelp)

	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(openCmd)
}

func dashHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "dash" {
		if defaultDashHelp != nil {
			defaultDashHelp(cmd, args)
		}
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ dash")+" "+ui.Faint("— visual dashboard"),
		[]ui.HelpGroup{
			{Title: "Commands", Commands: []ui.HelpEntry{
				{Name: "start", Desc: "Start the dashboard server"},
				{Name: "open", Desc: "Open in browser"},
			}},
		},
		"spwn dash [command]",
		"Use \"spwn dash <command> --help\" for more information.",
	)
}
