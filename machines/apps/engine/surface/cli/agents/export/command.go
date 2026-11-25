package agentsexport

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

type Input struct{ Name, To string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "export <agent> --to <folder>",
		Short:   "Write an agent's Mind into one archive, the Mind left as it is",
		Example: "  station agents export neo --to ~/Backups",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.To, "to", ".", "the folder the archive is written into")
		},
		Bind: func(in *Input, args []string) error { in.Name = args[0]; return nil },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			project, err := d.Project(ctx, env.Target())
			if err != nil {
				return err
			}
			x, err := call.Invoke(ctx, env.Calls, env.Target(), agentscalls.Export, agentscalls.ExportIn{Project: project, Agent: in.Name, To: in.To}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageagents.Done("agents-export", "agents export", x.Agent, "exported", x.Archive, d.ThisMachine), nil)
		},
	}, env)
}
