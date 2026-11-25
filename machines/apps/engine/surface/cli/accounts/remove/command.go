package accountsremove

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	grantscalls "github.com/jterrazz/studio-station/apps/engine/grants/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageaccounts "github.com/jterrazz/studio-station/apps/engine/pages/accounts"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
	vaultcalls "github.com/jterrazz/studio-station/apps/engine/vault/calls"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct{ Entry string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "remove <entry>",
		Short:   "Take a password out of a machine: a VM's grant of it ends, or on a Mac it leaves the vault with every grant made from it",
		Example: "  station accounts remove <entry> --on linux-1 --yes\n  station accounts remove <entry> --yes",
		Args:    cobra.ExactArgs(1),
		Bind:    func(in *Input, args []string) error { in.Entry = args[0]; return nil },
		Asks:    true,
		Confirm: func(in *Input) string { return "Remove " + in.Entry + "?" },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			// invariant: on a VM the password is its grant, on a Mac its
			// vault's entry: remove takes it out of the one the target holds.
			aim := kit.Aim(ctx, env, d.ThisMachine.Machine)
			e, err := vaultcalls.EntryNamed(ctx, env.Calls, aim.At, in.Entry)
			if err != nil {
				return err
			}
			if aim.VM != "" {
				var revoked accounts.Revoked
				err = io.Confirmed(func(answer string) (err error) {
					revoked, err = call.Invoke(ctx, env.Calls, aim.At, grantscalls.GrantRemove, grantscalls.GrantRemoveIn{Entry: e.ID, To: aim.VM, Confirm: answer}, nil)
					return err
				})
				if err != nil {
					return err
				}
				return io.Page(pageaccounts.Revoked(e.Label, aim.VM, revoked, d.ThisMachine), nil)
			}
			var removed accounts.Removed
			err = io.Confirmed(func(answer string) (err error) {
				removed, err = call.Invoke(ctx, env.Calls, aim.At, vaultcalls.VaultRemove, vaultcalls.VaultRemoveIn{ID: e.ID, Confirm: answer}, nil)
				return err
			})
			if err != nil {
				return err
			}
			return io.Page(pageaccounts.Forgotten(e.Label, aim.Name, len(removed.Removed), d.ThisMachine), nil)
		},
	}, env)
}
