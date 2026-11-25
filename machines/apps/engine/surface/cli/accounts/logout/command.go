package accountslogout

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageaccounts "github.com/jterrazz/studio-station/apps/engine/pages/accounts"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct{ Program string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "logout <program>",
		Short:   "Sign a program out on a machine, by its own logout: its token stays valid at its provider",
		Example: "  station accounts logout <program>\n  station accounts logout <program> --on linux-1 --yes",
		Args:    cobra.ExactArgs(1),
		Bind:    func(in *Input, args []string) error { in.Program = args[0]; return nil },
		Asks:    true,
		Confirm: func(in *Input) string { return "Sign " + accounts.Program(in.Program).Label() + " out?" },
		Verb:    func(*Input) string { return "sign out" },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			aim := kit.Aim(ctx, env, d.ThisMachine.Machine)
			var row accounts.ProgramRow
			err := io.Confirmed(func(answer string) (err error) {
				row, err = call.Invoke(ctx, env.Calls, aim.At, accountscalls.Logout, accountscalls.LogoutIn{Machine: aim.VM, Program: accounts.Program(in.Program), Confirm: answer}, nil)
				return err
			})
			if err != nil {
				return err
			}
			m := pageaccounts.Machine{Name: aim.Name, Kind: string(aim.Kind), Read: accounts.ProgramsRead{Programs: []accounts.ProgramRow{row}}}
			return io.Page(pageaccounts.Page(m, d.ThisMachine), nil)
		},
	}, env)
}
