package agentsmergepr

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

type Input struct{ Session string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "merge-pr <session>",
		Short:   "Merge the PR an agent opened, with the workspace's own GitHub account: the agent is finished",
		Example: "  station agents merge-pr s-1\n  station agents merge-pr s-1 --on linux-1",
		Args:    cobra.ExactArgs(1),
		Bind:    func(in *Input, args []string) error { in.Session = args[0]; return nil },
		Run: func(ctx context.Context, in *Input, o kit.IO) error {
			a, err := call.Invoke(ctx, env.Calls, env.Target(), agentscalls.MergePR, agentscalls.MergeIn{Session: in.Session}, nil)
			if err != nil {
				return err
			}
			return o.Page(pageagents.Merged(a, d.ThisMachine), a)
		},
	}, env)
}
