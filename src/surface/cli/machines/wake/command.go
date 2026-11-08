package machineswake

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/page"
	pagemachines "github.com/jterrazz/studio-station/src/pages/machines"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	Names       func() []string
	ThisMachine page.Context
}

type Input struct {
	Name string
	Tell bool
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "wake <mac>",
		Short:   "Wake a Mac that sleeps: a Wake-on-LAN packet sent from this Mac",
		Example: "  station machines wake macbook-air\n  station machines wake macbook-air --tell",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.BoolVar(&in.Tell, "tell", false, "tell me once it answers again")
		},
		Bind: func(in *Input, args []string) error {
			in.Name = args[0]
			return nil
		},
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			w, err := call.Invoke(ctx, env.Calls, machine.Ref{}, machinescalls.Wake, machinescalls.WakeIn{Name: in.Name, Tell: in.Tell}, nil)
			if err != nil {
				return err
			}
			return io.Page(pagemachines.Woken(w, d.ThisMachine), w)
		},
	}, env)
}
