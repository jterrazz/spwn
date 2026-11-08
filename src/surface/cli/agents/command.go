package agentscmd

import (
	"context"

	"github.com/spf13/cobra"

	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/page"
	pageagents "github.com/jterrazz/studio-station/src/pages/agents"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

func At(env kit.Env) machine.Ref {
	// value: where a command asks agents.list and agents.start: on a Mac this
	// Mac, which reaches every machine; inside a workspace its target.
	if env.Kind == machine.Mac {
		return env.Self
	}
	return env.Target()
}

func Scope(env kit.Env) agentscalls.ListIn {
	// value: the read of agents.list a command asks where it runs: the
	// workspace --on names, else every machine's from a Mac, a workspace's
	// own inside it.
	if env.Far() {
		return agentscalls.ListIn{On: env.Target().Name}
	}
	return agentscalls.ListIn{}
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
			kept, err := call.Invoke(ctx, env.Calls, At(env), agentscalls.List, Scope(env), nil)
			if err != nil {
				return err
			}
			if env.Far() || env.Kind != machine.Mac {
				return io.Page(pageagents.List(kept, d.ThisMachine), kept)
			}
			return io.Page(pageagents.AllMachines(kept, d.ThisMachine, env.Now()), kept)
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}
