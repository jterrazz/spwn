package machinesunlock

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
		Use:     "unlock <mac>",
		Short:   "Unlock a Mac's disk after a power cut: you type its password into macOS's own prompt",
		Example: "  station machines unlock mac-mini",
		Args:    cobra.ExactArgs(1),
		Bind: func(in *Input, args []string) error {
			in.Name = args[0]
			return nil
		},
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			u, err := call.Invoke(ctx, env.Calls, machine.Ref{}, machinescalls.Unlock, machinescalls.UnlockIn{Name: in.Name, Interactive: io.Interactive}, nil)
			if err != nil {
				return err
			}
			return io.Page(pagemachines.Unlocked(u, d.ThisMachine), nil)
		},
	}, env)
}
