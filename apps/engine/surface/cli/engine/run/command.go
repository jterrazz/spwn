package enginerun

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Engine interface {
	Check() error
	Run(ctx context.Context, note func(string)) error
}

type Deps struct{ Engine Engine }

func Command(d Deps, env kit.Env) *cobra.Command {
	// invariant: plumbing for this Mac's one launchd agent: it runs until the
	// agent stops it, and answers on its socket, never on stdout.
	return kit.Command(kit.Leaf[struct{}]{
		Use:      "run",
		Short:    "Run this Mac's engine: it supervises the VMs and shares their ports",
		Plumbing: true,
		Validate: func(context.Context, *struct{}) error { return d.Engine.Check() },
		Run:      func(ctx context.Context, _ *struct{}, io kit.IO) error { return d.Engine.Run(ctx, io.Note) },
	}, env)
}
