package observatory

import (
	"fmt"

	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

// Cmd is the parent command for Observatory operations.
var Cmd = &cobra.Command{
	Use:   "observatory",
	Short: "Visual dashboard for monitoring worlds and agents",
	Long:  `The Observatory is the eye of God — a real-time visual dashboard showing all worlds, agents, and their evolution.`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Observatory API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := universe.NewStore()
		if err != nil {
			return err
		}
		srv := universe.NewObservatoryServer(store, ":3001")
		fmt.Println("  Observatory API on http://localhost:3001")
		return srv.Start()
	},
}

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open the Observatory in your browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("  (Not yet implemented — coming in Epoch 7)")
		return nil
	},
}

func init() {
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(openCmd)
}
