package drift

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachinefile "github.com/jterrazz/studio-station/apps/engine/pages/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct{ ThisMachine page.Context }

type Input struct{ Machinefile string }

func Command(deps Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "drift",
		Short:   "How this machine differs from its Machinefile: each package present, absent, outdated, changed, not managed, extra",
		Example: "  station drift\n  station drift --json",
		Group:   kit.StationItself,
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Machinefile, "machinefile", "", "the Machinefile read, by its name or its file (default: the one this machine carries)")
		},
		Read:  func(*Input) bool { return true },
		Pages: true,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			d, err := call.Invoke(ctx, env.Calls, env.Target(), machinefilecalls.Drift, machinefilecalls.PlanIn{Machinefile: in.Machinefile}, nil)
			if err != nil {
				return err
			}
			return io.Page(pagemachinefile.Drift(d, deps.ThisMachine), pagemachinefile.PlanContract(machinefilecalls.Drift.Schema(), d.Plan))
		},
	}, env)
}
