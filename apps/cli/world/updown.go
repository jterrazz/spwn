package world

import (
	"github.com/spf13/cobra"
)

// DownCmd is `spwn down`. With no positional arg and a spwn project
// active, it stops every running world whose config name matches an
// entry in spwn.yaml (compose-style). With an explicit name or ID, it
// forwards to destroyCmd.
var DownCmd = &cobra.Command{
	Use:   "down [name-or-id]",
	Short: "Destroy a world",
	Args:  cobra.MaximumNArgs(1),
	RunE:  composeDownRunE,
}

func init() {
	registerSpawnFlags(UpCmd)
	DownCmd.Flags().BoolVar(&destroyAll, "all", false, "Destroy all running worlds")
}
