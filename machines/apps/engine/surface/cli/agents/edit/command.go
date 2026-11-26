package agentsedit

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
		Use:     "edit <agent>",
		Short:   "Change an agent's team or harness; an empty value clears it",
		Example: "  station agents edit neo --team web\n  station agents edit neo --team \"\"",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			flags = f
			f.StringVar(&in.Team, "team", "", "the team it belongs to, by slug; empty for none")
			f.StringVar(&in.Harness, "harness", "", "the harness it runs in; empty for its project's default")
		},
		Bind: func(in *Input, args []string) error { in.Name = args[0]; return nil },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			c := agentscalls.EditIn{Agent: in.Name}
			if flags.Changed("team") {
				c.Team = &in.Team
			}
			if flags.Changed("harness") {
				c.Harness = &in.Harness
			}
			project, err := d.Project(ctx, env.Target())
			if err != nil {
				return err
			}
			c.Project = project
			a, err := call.Invoke(ctx, env.Calls, env.Target(), agentscalls.Edit, c, nil)
			if err != nil {
				return err
			}
			said := "team " + or(a.Team, "none") + ", harness " + or(a.Harness, "the project's default")
			return io.Page(pageagents.Done("agents-edit", "agents edit", a.Name, said, "", d.ThisMachine), nil)
		},
	}, env)
}

func or(v, none string) string {
	if v == "" {
		return none
	}
	return v
}
