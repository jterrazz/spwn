package machinesforget

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
		Use:     "forget <mac>",
		Short:   "Forget another Mac here: its ssh block goes, nothing changes on it",
		Example: "  station machines forget mac-mini --yes",
		Args:    cobra.ExactArgs(1),
		Bind: func(in *Input, args []string) error {
			in.Name = args[0]
			return nil
		},
		Asks:     true,
		Confirm:  func(in *Input) string { return "Forget " + in.Name + "?" },
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			err := io.Confirmed(func(answer string) error {
				_, err := call.Invoke(ctx, env.Calls, machine.Ref{}, machinescalls.Forget, machinescalls.ForgetIn{Name: in.Name, Confirm: answer}, nil)
				return err
			})
			if err != nil {
				return err
			}
			return io.Page(pagemachines.Forgot(in.Name, d.ThisMachine), nil)
		},
	}, env)
}
