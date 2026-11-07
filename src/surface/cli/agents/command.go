package agentscmd

import (
	"context"

	"github.com/spf13/cobra"

	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/page"
	pageagents "github.com/jterrazz/studio-station/src/pages/agents"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

func Command(d Deps, env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Command(kit.Leaf[struct{}]{
		Use:     "agents",
		Short:   "The agents working in this workspace: waiting on you, working, finished",
		Example: "  station agents\n  station agents --on linux-1\n  station agents --json",
		Read:    func(*struct{}) bool { return true },
		Pages:   true,
		Run: func(ctx context.Context, _ *struct{}, io kit.IO) error {
			kept, err := call.Invoke(ctx, env.Calls, env.Target(), agentscalls.List, struct{}{}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageagents.List(kept, d.ThisMachine), kept)
		},
	}, env)
	cmd.Hidden = true
	cmd.AddCommand(children...)
	return cmd
}
