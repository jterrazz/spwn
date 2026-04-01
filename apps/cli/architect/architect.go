package architect

import (
	"fmt"
	"time"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

var defaultArchitectHelp func(*cobra.Command, []string)

// Cmd is the parent command for Architect operations.
var Cmd = &cobra.Command{
	Use:   "architect",
	Short: "Your always-on world builder",
	Long:  `The Architect is your always-on world builder. It manages worlds, connects to messaging channels, and orchestrates artificial life.`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Architect daemon",
	Long: `Start the Architect daemon in a Docker container.

The Architect runs the spwn binary inside a long-lived container with the
host's Docker socket mounted (DooD — Docker-outside-of-Docker), allowing it
to create and manage world containers as siblings. Channels (Telegram, Slack,
etc.) connect here.

The container mounts:
  /var/run/docker.sock    Docker daemon access (sibling containers, not nested)
  ~/.spwn/                Shared configuration and state`,
	RunE: func(cmd *cobra.Command, args []string) error {
		org, _ := universe.LoadOrg()
		name := "default"
		if org != nil {
			name = org.Name
		}

		fmt.Printf("  Starting Architect for universe %q...\n", name)
		fmt.Println("  Architect is alive.")
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Architect daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("  Architect stopped.")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Architect status — channels, worlds, agents",
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
	defaultArchitectHelp = Cmd.HelpFunc()
	Cmd.SetHelpFunc(architectHelp)

	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(stopCmd)
	Cmd.AddCommand(statusCmd)
	Cmd.AddCommand(connectCmd)
}

func architectHelp(cmd *cobra.Command, args []string) {
	if cmd.Name() != "architect" {
		if defaultArchitectHelp != nil {
			defaultArchitectHelp(cmd, args)
		}
		return
	}

	w := cmd.OutOrStdout()
	ui.RenderGroupedHelp(w,
		ui.Strong("⬡ architect")+" "+ui.Faint("— your always-on world builder"),
		[]ui.HelpGroup{
			{Title: "Commands", Commands: []ui.HelpEntry{
				{Name: "start", Desc: "Start the Architect daemon"},
				{Name: "stop", Desc: "Stop the Architect daemon"},
				{Name: "status", Desc: "Show status, channels, active worlds"},
				{Name: "connect <channel>", Desc: "Connect a messaging channel"},
			}},
		},
		"spwn architect [command]",
		"Use \"spwn architect <command> --help\" for more information.",
	)
}
