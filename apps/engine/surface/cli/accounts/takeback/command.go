package accountstakeback

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	pageaccounts "github.com/jterrazz/studio-station/apps/engine/pages/accounts"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Input struct{ Account, From string }

func bind(in *Input, args []string) error {
	in.Account = args[0]
	if in.From == "" {
		return kit.Refuse("station accounts take-back needs a VM", "station accounts take-back "+in.Account+" --from <vm>")
	}
	return nil
}

func Command(env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "take-back <account> --from <vm> [--yes]",
		Short:   "Delete the copy of an account's secret a VM holds",
		Example: "  station accounts take-back github --from linux-1",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.From, "from", "", "the VM that holds the copy")
		},
		Bind:    bind,
		Asks:    true,
		Confirm: func(in *Input) string { return "Take " + accounts.Label(in.Account) + " back from " + in.From + "?" },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			var out accounts.Shared
			err := io.Confirmed(func(answer string) (err error) {
				out, err = call.Invoke(ctx, env.Calls, env.Target(), accountscalls.TakeBack, accountscalls.TakeBackIn{Account: in.Account, From: in.From, Confirm: answer}, nil)
				return err
			})
			if err != nil {
				return err
			}
			return io.Page(pageaccounts.TakenBack(out), nil)
		},
	}, env)
}

type TakeBackInput struct {
	From  string
	Names []string
}

func RecallAlias(env kit.Env) *cobra.Command {
	// value: secrets recall, the v2 words of take-back: each secret named
	// (every one the VM holds, when none is) is the account it carries.
	return kit.Command(kit.Leaf[TakeBackInput]{
		Use:  "recall --from <vm> [<name...>]",
		Args: cobra.ArbitraryArgs,
		Declare: func(f *pflag.FlagSet, in *TakeBackInput) {
			f.StringVar(&in.From, "from", "", "the VM that holds the copies")
		},
		Bind: func(in *TakeBackInput, args []string) error {
			in.Names = args
			if in.From == "" {
				return kit.Refuse("station accounts take-back needs a VM", "station accounts take-back <account> --from <vm>")
			}
			return nil
		},
		Asks:    true,
		Confirm: func(in *TakeBackInput) string { return "Take the shares back from " + in.From + "?" },
		Run: func(ctx context.Context, in *TakeBackInput, io kit.IO) error {
			names := in.Names
			if len(names) == 0 {
				r, err := call.Read(ctx, env.Calls, env.Target(), accountscalls.List, accountscalls.ListIn{}, nil)
				if err != nil {
					return err
				}
				for _, sh := range r.Shares {
					if sh.VM == in.From {
						names = append(names, sh.Account)
					}
				}
			}
			for _, name := range names {
				var out accounts.Shared
				err := io.Confirmed(func(answer string) (err error) {
					out, err = call.Invoke(ctx, env.Calls, env.Target(), accountscalls.TakeBack, accountscalls.TakeBackIn{Account: accounts.AccountOf(name), From: in.From, Confirm: answer}, nil)
					return err
				})
				if err != nil {
					return err
				}
				if err := io.Page(pageaccounts.TakenBack(out), nil); err != nil {
					return err
				}
			}
			return nil
		},
	}, env)
}
