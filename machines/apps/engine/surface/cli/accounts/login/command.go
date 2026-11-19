package accountslogin

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
		Use:     "login <program>",
		Short:   "Sign a program in on a machine: its code or link is printed, and you finish at its site",
		Example: "  station accounts login <program>\n  station accounts login <program> --on linux-1",
		Args:    cobra.ExactArgs(1),
		Bind:    func(in *Input, args []string) error { in.Program = args[0]; return nil },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			if !io.Interactive {
				return kit.NeedsTerminal("station accounts login")
			}
			aim := kit.Aim(ctx, env, d.ThisMachine.Machine)
			updates, err := call.Follow(ctx, env.Calls, aim.At, accountscalls.Login, accountscalls.LoginIn{Machine: aim.VM, Program: accounts.Program(in.Program)}, nil)
			if err != nil {
				return err
			}
			for u := range updates {
				if u.Err != nil {
					return u.Err
				}
				if s := u.Value.Step; s != nil {
					if err := io.Page(pageaccounts.LoginStep(*s, d.ThisMachine), nil); err != nil {
						return err
					}
				}
				if len(u.Value.Programs) > 0 {
					m := pageaccounts.Machine{Name: aim.Name, Kind: string(aim.Kind), Read: accounts.ProgramsRead{Programs: u.Value.Programs}}
					if err := io.Page(pageaccounts.Page(m, d.ThisMachine), nil); err != nil {
						return err
					}
				}
			}
			return ctx.Err()
		},
	}, env)
}
