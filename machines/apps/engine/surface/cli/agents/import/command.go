package agentsimport

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

type Input struct{ Archive, As string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "import <archive>",
		Short:   "Make an agent from an archive an export wrote, named as the archive unless --as says",
		Example: "  station agents import neo.tar.gz\n  station agents import neo.tar.gz --as trinity",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.As, "as", "", "the name the agent takes, instead of the archive's")
		},
		Bind: func(in *Input, args []string) error { in.Archive = args[0]; return nil },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			project, err := d.Project(ctx, env.Target())
			if err != nil {
				return err
			}
			m, err := call.Invoke(ctx, env.Calls, env.Target(), agentscalls.Import, agentscalls.ImportIn{Project: project, Archive: in.Archive, As: in.As}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageagents.Done("agents-import", "agents import", m.Name, "imported", m.Path, d.ThisMachine), nil)
		},
	}, env)
}
