package automationslog

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	automationscalls "github.com/jterrazz/studio-station/apps/engine/automations/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageautomations "github.com/jterrazz/studio-station/apps/engine/pages/automations"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	Project     func(ctx context.Context, target machine.Ref) (string, error)
}

type Input struct {
	Name string
	Last int
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "log [<automation>]",
		Short:   "The runs written for this folder's project, oldest first; one automation's when it is named",
		Example: "  station automations log\n  station automations log morning --last 5\n  station automations log --json",
		Args:    cobra.MaximumNArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.IntVar(&in.Last, "last", 0, "how many of the newest runs to show (default 50)")
		},
		Bind: func(in *Input, args []string) error {
			if len(args) == 1 {
				in.Name = args[0]
			}
			return nil
		},
		Read:  func(*Input) bool { return true },
		Pages: true,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			project, err := d.Project(ctx, env.Target())
			if err != nil {
				return err
			}
			runs, err := call.Invoke(ctx, env.Calls, env.Target(), automationscalls.Log, automationscalls.LogIn{Project: project, Automation: in.Name, Last: in.Last}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageautomations.Log(runs, d.ThisMachine), pageautomations.LogContract(automationscalls.Log.Schema(), runs))
		},
	}, env)
}
