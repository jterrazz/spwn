package agentsmerge

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
	// value: the workspace the calls for the agent of that name go to, and
	// the name it knows the agent by.
	Where func(ctx context.Context, name string) (machine.Ref, string, error)
}

type Input struct{ Agent string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "merge <agent>",
		Short:   "Merge the PR an agent opened, with its workspace's own GitHub account: the agent is finished",
		Example: "  station agents merge review-bot",
		Args:    cobra.ExactArgs(1),
		Bind:    func(in *Input, args []string) error { in.Agent = args[0]; return nil },
		Run: func(ctx context.Context, in *Input, o kit.IO) error {
			at, local, err := d.Where(ctx, in.Agent)
			if err != nil {
				return err
			}
			a, err := call.Invoke(ctx, env.Calls, at, agentscalls.Merge, agentscalls.AgentIn{Agent: local}, nil)
			if err != nil {
				return err
			}
			a.Name = in.Agent
			return o.Page(pageagents.Merged(a, d.ThisMachine), a)
		},
	}, env)
}
