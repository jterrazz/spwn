package world

import (
	"github.com/spf13/cobra"
)

// UpCmd is the top-level alias for `spwn up` (= `spwn world up`).
var UpCmd = &cobra.Command{
	Use:     "up",
	Short:   "Spawn a world — an isolated reality for agents",
	Long:    upCmd.Long,
	Example: upCmd.Example,
	RunE:    spawnRunE,
}

// DownCmd is the top-level alias for spwn down.
var DownCmd = &cobra.Command{
	Use:   "down [world-id]",
	Short: "Destroy a world",
	Args:  cobra.MaximumNArgs(1),
	RunE:  destroyCmd.RunE,
}

// LsCmd is the top-level alias for spwn ls.
var LsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List active worlds",
	RunE:  listCmd.RunE,
}

func init() {
	registerSpawnFlags(UpCmd)
	DownCmd.Flags().BoolVar(&destroyAll, "all", false, "Destroy all running worlds")
}
