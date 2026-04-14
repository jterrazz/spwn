package world

import (
	"github.com/spf13/cobra"
)

// startCmd is `spwn world start [name]` — an alias for `spwn world up`
// in the new compose-style grammar. Having an explicit `start` verb
// alongside `up` matches what the root help surfaces and mirrors the
// docker compose terminology.
var startCmd = &cobra.Command{
	Use:     "start [name]",
	Aliases: []string{"up"},
	Short:   "Start a world (alias for `spwn up`)",
	Args:    cobra.MaximumNArgs(1),
	RunE:    composeUpRunE,
}

// stopCmd is `spwn world stop [name-or-id]` — an alias for
// `spwn world down`. Stopping a world keeps its agents: their minds
// live in spwn/agents/<name>/ and will rehydrate on the next start.
var stopCmd = &cobra.Command{
	Use:     "stop [name-or-id]",
	Aliases: []string{"down"},
	Short:   "Stop a world (alias for `spwn down`)",
	Args:    cobra.MaximumNArgs(1),
	RunE:    composeDownRunE,
}

func init() {
	// Register both verbs under `spwn world ...`. They reuse the same
	// RunE as the top-level aliases so behavior stays in lockstep.
	registerSpawnFlags(startCmd)
	stopCmd.Flags().BoolVar(&destroyAll, "all", false, "Destroy all running worlds")
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(stopCmd)
}
