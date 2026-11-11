package plan

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/core/call"
	machinefilecalls "github.com/jterrazz/studio-station/src/machinefile/calls"
	"github.com/jterrazz/studio-station/src/page"
	pagemachinefile "github.com/jterrazz/studio-station/src/pages/machinefile"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct{ ThisMachine page.Context }

type Input struct{ Machinefile string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "plan",
		Short:   "What station apply would do here: what it asks of you, installs, sets and adopts",
		Example: "  station plan\n  station plan --machinefile station-macos",
		Group:   kit.StationItself,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Machinefile, "machinefile", "", "the Machinefile planned, by its name or its file (default: the one this machine carries)")
		},
		Read:  func(*Input) bool { return true },
		Pages: true,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			p, err := call.Invoke(ctx, env.Calls, env.Target(), machinefilecalls.Plan, machinefilecalls.PlanIn{Machinefile: in.Machinefile}, nil)
			if err != nil {
				return err
			}
			return io.Page(pagemachinefile.Plan(p, d.ThisMachine), pagemachinefile.PlanContract("station.machinefile.plan/v1", p))
		},
	}, env)
}
