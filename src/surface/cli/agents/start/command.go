package agentsstart

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/agents"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/page"
	pageagents "github.com/jterrazz/studio-station/src/pages/agents"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct {
	App, Folder, Task string
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "start [folder]",
		Short:   "Start an agent in this workspace's terminal session, its task as its first prompt",
		Example: "  station agents start ~/app --task 'Build the landing page'\n  station agents start --on linux-1",
		Args:    cobra.MaximumNArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.App, "app", "", "the agent's app, by its adapter's name; the first station reads by default")
			f.StringVar(&in.Task, "task", "", "its first prompt")
		},
		Bind: func(in *Input, args []string) error {
			if len(args) > 0 {
				in.Folder = args[0]
			}
			return nil
		},
		Run: func(ctx context.Context, in *Input, o kit.IO) error {
			s, err := call.Invoke(ctx, env.Calls, env.Target(), agentscalls.Start, agentscalls.StartIn{App: agents.App(in.App), Folder: in.Folder, Task: in.Task}, nil)
			if err != nil {
				return err
			}
			return o.Page(pageagents.Started(s.App, s.Folder, s.Task, d.ThisMachine), s)
		},
	}, env)
}
