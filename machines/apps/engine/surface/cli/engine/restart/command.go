package enginerestart

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageengine "github.com/jterrazz/studio-station/apps/engine/pages/engine"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[struct{}]{
		Use:     "restart",
		Short:   "Restart this Mac's engine; the VMs that run keep running",
		Example: "  station engine restart",
		Run: func(ctx context.Context, _ *struct{}, io kit.IO) error {
			report, err := call.Invoke(ctx, env.Calls, env.Target(), enginecalls.Restart, enginecalls.RestartIn{}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageengine.Restart(report, d.ThisMachine), nil)
		},
	}, env)
}
