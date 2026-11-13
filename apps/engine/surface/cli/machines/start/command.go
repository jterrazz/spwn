package machinesstart

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachines "github.com/jterrazz/studio-station/apps/engine/pages/machines"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	// value: the machines known here, for completion.
	Names       func() []string
	ThisMachine page.Context
}

type Input struct{ Name string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "start <vm>",
		Short:   "Start a stopped VM, with its disk as it was left; a server is always on again",
		Example: "  station machines start api",
		Args:    cobra.ExactArgs(1),
		Bind: func(in *Input, args []string) error {
			in.Name = args[0]
			return nil
		},
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			stop := io.Wait("starting " + in.Name)
			out, err := call.Invoke(ctx, env.Calls, env.Target(), machinescalls.Start, machinescalls.VMIn{VM: in.Name}, nil)
			stop()
			if err != nil {
				return kit.OnItsMac(ctx, env, in.Name, err)
			}
			if out.Already {
				return io.Page(pagemachines.Already("start", in.Name, true, d.ThisMachine), nil)
			}
			if out.Kind == machine.Server {
				return io.Page(pagemachines.ServerStarted(in.Name, d.ThisMachine), nil)
			}
			return io.Page(pagemachines.Change("start", in.Name, "started", d.ThisMachine), nil)
		},
	}, env)
}
