package accountsshare

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/accounts"
	accountscalls "github.com/jterrazz/studio-station/src/accounts/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	pageaccounts "github.com/jterrazz/studio-station/src/pages/accounts"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	Now func() time.Time
}

type Input struct {
	Account, With, For string
	Term               time.Duration
}

func bind(in *Input, args []string) error {
	in.Account = args[0]
	if in.With == "" {
		return kit.Refuse("station accounts share needs a VM", "station accounts share "+in.Account+" --with <vm>")
	}
	in.Term = accounts.DefaultShareTerm
	if in.For == "" {
		return nil
	}
	term, ok := accounts.ParseTerm(in.For)
	if !ok {
		return kit.Refuse("\""+in.For+"\" is not a duration", "station accounts share "+in.Account+" --with "+in.With+" --for 12h")
	}
	in.Term = term
	return nil
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "share <account> --with <vm> [--for <duration>]",
		Short:   "Copy an account's secret into a VM of this Mac, for a time; every share ends by itself",
		Example: "  station accounts share github --with linux-1\n  station accounts share claude --with linux-1 --for 4h",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.With, "with", "", "the VM that receives the copy")
			f.StringVar(&in.For, "for", "", "how long the share lasts, 1m to 7d (default 12h)")
		},
		Bind: bind,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			// value: a VM another Mac holds is asked of that Mac, which says
			// whether this Mac may share with it.
			if _, inVM := kit.Holder(ctx, env, env.Target().Name); !inVM {
				if h, ok := kit.Holder(ctx, env, in.With); ok && !h.OnTarget(env) {
					return shareAt(ctx, env, d, in, machine.Ref{Name: h.Mac}, env.Target().Name, io)
				}
			}
			return shareAt(ctx, env, d, in, env.Target(), "", io)
		},
	}, env)
}

type ShareInput struct {
	To    string
	Names []string
}

func LendAlias(d Deps, env kit.Env) *cobra.Command {
	// value: secrets lend, the v2 words of share: --to is --with, and each
	// secret named (every one the share list holds, when none is) is the
	// account it carries.
	cmd := kit.Command(kit.Leaf[ShareInput]{
		Use:  "lend --to <vm> [<name...>]",
		Args: cobra.ArbitraryArgs,
		Declare: func(f *pflag.FlagSet, in *ShareInput) {
			f.StringVar(&in.To, "with", "", "the VM that receives the copies")
		},
		Bind: func(in *ShareInput, args []string) error {
			in.Names = args
			if in.To == "" {
				return kit.Refuse("station accounts share needs a VM", "station accounts share <account> --with <vm>")
			}
			return nil
		},
		Run: func(ctx context.Context, in *ShareInput, io kit.IO) error {
			names := in.Names
			if len(names) == 0 {
				r, err := call.Read(ctx, env.Calls, env.Target(), accountscalls.List, accountscalls.ListIn{}, nil)
				if err != nil {
					return err
				}
				names = accounts.Lendable(r.ShareList, "")
			}
			for _, name := range names {
				shared, err := call.Invoke(ctx, env.Calls, env.Target(), accountscalls.Share, accountscalls.ShareIn{Account: accounts.AccountOf(name), With: in.To}, nil)
				if err != nil {
					return err
				}
				if err := io.Page(pageaccounts.Shared(shared, d.Now()), nil); err != nil {
					return err
				}
			}
			return nil
		},
	}, env)
	cmd.Flags().SetNormalizeFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
		if name == "to" {
			name = "with"
		}
		return pflag.NormalizedName(name)
	})
	return cmd
}

func shareAt(ctx context.Context, env kit.Env, d Deps, in *Input, at machine.Ref, from string, io kit.IO) error {
	shared, err := call.Invoke(ctx, env.Calls, at, accountscalls.Share, accountscalls.ShareIn{Account: in.Account, With: in.With, For: in.Term, From: from}, nil)
	if err != nil {
		return err
	}
	return io.Page(pageaccounts.Shared(shared, d.Now()), nil)
}
