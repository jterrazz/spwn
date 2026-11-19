package machinesremind

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
	Names       func() []string
	ThisMachine page.Context
}

type Input struct{ Name string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "remind <mac>",
		Short:   "Tell me once a Mac that sleeps answers again, nothing sent to it",
		Example: "  station machines remind macbook-air",
		Args:    cobra.ExactArgs(1),
		Bind: func(in *Input, args []string) error {
			in.Name = args[0]
			return nil
		},
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			r, err := call.Invoke(ctx, env.Calls, machine.Ref{}, machinescalls.Remind, machinescalls.RemindIn{Name: in.Name}, nil)
			if err != nil {
				return err
			}
			return io.Page(pagemachines.Reminded(r, d.ThisMachine), r)
		},
	}, env)
}
