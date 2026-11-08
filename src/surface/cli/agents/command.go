package agentscmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/agents"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/page"
	pageagents "github.com/jterrazz/studio-station/src/pages/agents"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	// value: every agent of every machine: those of each running workspace
	// of each Mac that answers.
	Everywhere func(ctx context.Context) ([]agents.Agent, error)
}

func Command(d Deps, env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Command(kit.Leaf[struct{}]{
		Use:     "agents",
		Short:   "Every agent of every machine: waiting on you, working, finished, and the workspace each runs on",
		Example: "  station agents\n  station agents --on linux-1\n  station agents --json",
		Group:   kit.Machines,
		Read:    func(*struct{}) bool { return true },
		Pages:   true,
		Run: func(ctx context.Context, _ *struct{}, io kit.IO) error {
			// invariant: SPEC B5.22: on a Mac, every machine's agents; with --on, or
			// inside a workspace, that workspace's own.
			if env.Far() || env.Kind != machine.Mac {
				kept, err := call.Invoke(ctx, env.Calls, env.Target(), agentscalls.List, agentscalls.ListIn{}, nil)
				if err != nil {
					return err
				}
				return io.Page(pageagents.List(kept, d.ThisMachine), kept)
			}
			kept, err := d.Everywhere(ctx)
			if err != nil {
				return err
			}
			return io.Page(pageagents.AllMachines(kept, d.ThisMachine, env.Now()), kept)
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}
