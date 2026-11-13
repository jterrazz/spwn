package machinessupervise

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

func Command(env kit.Env) *cobra.Command {
	// invariant: a round-3 per-VM agent that launchd restarts after an upgrade
	// ends at once, exit 0, instead of looping; it touches nothing.
	cmd := kit.Command(kit.Leaf[struct{}]{
		Use:     "supervise <vm>",
		Short:   "Retired: the engine supervises VMs",
		Args:    cobra.ArbitraryArgs,
		RawArgs: true,
		Run: func(_ context.Context, _ *struct{}, io kit.IO) error {
			io.Note(machines.SuperviseRetired)
			return nil
		},
	}, env)
	cmd.Hidden = true
	return cmd
}
