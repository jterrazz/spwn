package world

import (
	"github.com/spf13/cobra"
)

// UpCmd is the top-level alias for `spwn up` (= `spwn world up`).
var UpCmd = &cobra.Command{
	Use:     "up [name]",
	Short:   "Spawn a world - an isolated reality for agents",
	Long:    upCmd.Long,
	Example: upCmd.Example,
	Args:    cobra.MaximumNArgs(1),
	RunE:    composeUpRunE,
}

// DownCmd is the top-level alias for spwn down. With no positional arg
// and a spwn project active, it stops every running world whose config
// name matches an entry in spwn.yaml (compose-style). With an explicit
// name or ID, it forwards to destroyCmd.
var DownCmd = &cobra.Command{
	Use:   "down [name-or-id]",
	Short: "Destroy a world",
	Args:  cobra.MaximumNArgs(1),
	RunE:  composeDownRunE,
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
