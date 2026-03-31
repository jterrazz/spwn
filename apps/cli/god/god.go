package god

import (
	"fmt"
	"time"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

var defaultGodHelp func(*cobra.Command, []string)

// Cmd is the parent command for God operations.
var Cmd = &cobra.Command{
	Use:   "god",
	Short: "The God — always-on orchestration daemon",
	Long:  `The God is the orchestration daemon of spwn. It manages worlds, connects to messaging channels, and orchestrates artificial life.`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the God daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		org, _ := universe.LoadOrg()
		name := "default"
		if org != nil {
			name = org.Name
		}

		fmt.Printf("  Starting God for universe %q...\n", name)
		fmt.Println("  God is alive.")
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the God daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("  God stopped.")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show God status — channels, worlds, agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		org, _ := universe.LoadOrg()
		if org != nil {
			fmt.Printf("  Universe: %s\n", org.Name)
		}
		fmt.Printf("  Status: idle\n")
		fmt.Printf("  Uptime: %s\n", time.Duration(0))
		return nil
	},
}

var connectCmd = &cobra.Command{
	Use:   "connect [channel]",
	Short: "Connect a messaging channel (telegram, slack, discord, ...)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		channel := args[0]
		fmt.Printf("  Channel %q connected.\n", channel)
		return nil
	},
}

func init() {
	defaultGodHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(godHelp)

	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(stopCmd)
	Cmd.AddCommand(statusCmd)
	Cmd.AddCommand(connectCmd)
}

func godHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "god" {
		if defaultGodHelp != nil {
			defaultGodHelp(cmd, args)
		}
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ god")+" "+ui.Faint("— always-on orchestration daemon"),
		[]ui.HelpGroup{
			{Title: "Commands", Commands: []ui.HelpEntry{
				{"start", "Start the God daemon"},
				{"stop", "Stop the God daemon"},
				{"status", "Show status, channels, active worlds"},
				{"connect <channel>", "Connect a messaging channel"},
			}},
		},
		"spwn god [command]",
		"Use \"spwn god <command> --help\" for more information.",
	)
}
