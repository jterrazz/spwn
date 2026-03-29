package claw

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"spwn.sh/core/universe"
)

// Cmd is the parent command for Claw operations.
var Cmd = &cobra.Command{
	Use:   "claw",
	Short: "Manage the Claw — the always-on orchestration daemon",
	Long:  `The Claw is the God-layer of spwn. It manages worlds, connects to messaging channels, and orchestrates artificial life.`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Claw daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load org manifest if it exists
		org, _ := universe.LoadOrg()
		name := "default"
		if org != nil {
			name = org.Name
		}

		fmt.Printf("  Starting Claw for organization %q...\n", name)
		fmt.Println("  Claw is alive.")
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Claw daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("  Claw stopped.")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Claw status — channels, worlds, agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		org, _ := universe.LoadOrg()
		if org != nil {
			fmt.Printf("  Organization: %s\n", org.Name)
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
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(stopCmd)
	Cmd.AddCommand(statusCmd)
	Cmd.AddCommand(connectCmd)
}
