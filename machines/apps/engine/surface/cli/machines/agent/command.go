package machinesagent

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	// value: serves until ctx ends; every refusal comes before the socket opens.
	Serve func(ctx context.Context) error
}

type Input struct{}

func Command(d Deps, env kit.Env) *cobra.Command {
	// invariant: plumbing for a Mac's launchd agent: ssh reaches it through
	// IdentityAgent, and nothing is printed while it serves.
	return kit.Command(kit.Leaf[Input]{
		Use:      "agent",
		Short:    "Serve this Mac's device key to ssh",
		Plumbing: true,
		Args:     cobra.NoArgs,
		Bind:     func(*Input, []string) error { return nil },
		Run: func(ctx context.Context, _ *Input, _ kit.IO) error {
			return d.Serve(ctx)
		},
	}, env)
}
