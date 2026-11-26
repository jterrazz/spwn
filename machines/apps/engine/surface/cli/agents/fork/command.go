package agentsfork

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

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

type Input struct{ Name, As string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "fork <agent> --as <name>",
		Short:   "Make a new agent from an agent: a copy of every layer of its Mind, its journal left behind",
		Example: "  station agents fork neo --as trinity",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.As, "as", "", "the new agent's name")
		},
		Bind: func(in *Input, args []string) error { in.Name = args[0]; return nil },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			project, err := d.Project(ctx, env.Target())
			if err != nil {
				return err
			}
			f, err := call.Invoke(ctx, env.Calls, env.Target(), agentscalls.Fork, agentscalls.ForkIn{Project: project, Agent: in.Name, As: in.As}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageagents.Done("agents-fork", "agents fork", f.To, "forked from "+f.From, "", d.ThisMachine), nil)
		},
	}, env)
}
