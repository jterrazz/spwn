package agentscreate

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

type Input struct{ Name, Team, Harness string }

func Command(d Deps, env kit.Env) *cobra.Command {
	var flags *pflag.FlagSet
	return kit.Command(kit.Leaf[Input]{
		Use:     "create <name>",
		Short:   "Make an agent: a blank Mind in this folder's project, or your own outside one",
		Example: "  station agents create neo\n  station agents create neo --team web",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			flags = f
			f.StringVar(&in.Team, "team", "", "the team it belongs to, by slug")
			f.StringVar(&in.Harness, "harness", "", "the harness it runs in, one of the catalog's; its first when absent")
		},
		Bind: func(in *Input, args []string) error { in.Name = args[0]; return nil },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			project, err := d.Project(ctx, env.Target())
			if err != nil {
				return err
			}
			c := agentscalls.CreateIn{Project: project, Agent: in.Name}
			if flags.Changed("team") {
				c.Team = &in.Team
			}
			if flags.Changed("harness") {
				c.Harness = &in.Harness
			}
			m, err := call.Invoke(ctx, env.Calls, env.Target(), agentscalls.Create, c, nil)
			if err != nil {
				return err
			}
			return io.Page(pageagents.Done("agents-create", "agents create", m.Name, "created", m.Path, d.ThisMachine), nil)
		},
	}, env)
}
