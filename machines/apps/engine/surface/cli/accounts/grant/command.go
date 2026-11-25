package accountsgrant

import (
	"context"

	"github.com/spf13/cobra"

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
		Use:     "grant <entry>",
		Short:   "Give a password of this Mac's vault to a VM, by its label, until it is removed; Touch ID asks first",
		Example: "  station accounts grant pw-1 --on linux-1",
		Args:    cobra.ExactArgs(1),
		Bind:    func(in *Input, args []string) error { in.Entry = args[0]; return nil },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			// invariant: the vault is the VM's Mac's, and so is the grant;
			// a Mac itself is given nothing, as the call refuses it.
			aim := kit.Aim(ctx, env, d.ThisMachine.Machine)
			e, err := vaultcalls.EntryNamed(ctx, env.Calls, aim.At, in.Entry)
			if err != nil {
				return err
			}
			if _, err := call.Invoke(ctx, env.Calls, aim.At, grantscalls.GrantAdd, grantscalls.GrantAddIn{Entry: e.ID, To: aim.Name}, nil); err != nil {
				return err
			}
			return io.Page(pageaccounts.Granted(e.Label, aim.Name, d.ThisMachine), nil)
		},
	}, env)
}
