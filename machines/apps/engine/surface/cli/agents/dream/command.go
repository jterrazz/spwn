package agentsdream

import (
	"context"

	"github.com/spf13/cobra"

	agentscalls "github.com/jterrazz/studio-station/apps/engine/agents/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageagents "github.com/jterrazz/studio-station/apps/engine/pages/agents"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	Project     func(ctx context.Context, target machine.Ref) (string, error)
}

type Input struct{ Name string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "dream <agent>",
		Short:   "Read an agent's journal and write what it learned into a playbook of its Mind",
		Example: "  station agents dream neo",
		Args:    cobra.ExactArgs(1),
		Bind:    func(in *Input, args []string) error { in.Name = args[0]; return nil },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			project, err := d.Project(ctx, env.Target())
			if err != nil {
				return err
			}
			r, err := call.Invoke(ctx, env.Calls, env.Target(), agentscalls.Dream, agentscalls.DreamIn{Project: project, Agent: in.Name}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageagents.Dreamed(r, d.ThisMachine), nil)
		},
	}, env)
}
