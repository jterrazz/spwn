package accountsadd

import (
	"context"
	"slices"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageaccounts "github.com/jterrazz/studio-station/apps/engine/pages/accounts"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
	vaultcalls "github.com/jterrazz/studio-station/apps/engine/vault/calls"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct {
	Program, Label, Kind string
	FromSignIn           bool
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "add <program>",
		Short:   "Keep a password in this Mac's vault: typed into a hidden prompt, or taken from this Mac's sign-in; never an argument",
		Example: "  station accounts add <program> --label \"GitHub deploy\"\n  station accounts add <program> --from-sign-in",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Label, "label", "", "the password's name in the vault (default: the program's, or the one its sign-in proposes)")
			f.StringVar(&in.Kind, "kind", "", "what a typed password is: token, fine-grained, api-key, oauth-token (default: the first the program takes)")
			f.BoolVar(&in.FromSignIn, "from-sign-in", false, "take the token of this Mac's own sign-in to the program, read from it and never shown")
		},
		Bind: func(in *Input, args []string) error { in.Program = args[0]; return nil },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			// invariant: a typed value is asked in a hidden field and never
			// read from an argument, so a typed password needs a terminal; a
			// password taken from a sign-in asks nothing.
			program := accounts.Program(in.Program)
			if !program.Known() {
				return accounts.Refusal(fail.NotFound, accounts.LoginUnknownProgram, in.Program)
			}
			if !in.FromSignIn && !io.Interactive {
				return kit.NeedsTerminal("station accounts add")
			}
			offer, err := call.Read(ctx, env.Calls, env.Self, vaultcalls.VaultNew, vaultcalls.VaultNewIn{}, nil)
			if err != nil {
				return err
			}
			add := vaultcalls.VaultAddIn{Label: in.Label, Program: program, From: accounts.FromTyped, Kind: accounts.KeyKind(in.Kind)}
			if in.FromSignIn {
				add.From = accounts.FromSignIn
				if i := slices.IndexFunc(offer.Captures, func(c accounts.Capture) bool { return c.Program == program }); i >= 0 && add.Label == "" {
					add.Label = offer.Captures[i].Label
				}
			} else {
				if i := slices.IndexFunc(offer.Programs, func(t accounts.TypedProgram) bool { return t.Program == program }); i >= 0 && add.Kind == "" {
					add.Kind = offer.Programs[i].Kind
				}
				fields := []page.Field{}
				if add.Label == "" {
					fields = append(fields, page.Field{Key: "label", Label: "Its name in the vault", Kind: page.FieldText, Default: program.Label(), Required: true})
				}
				fields = append(fields, page.Field{Key: "value", Label: program.Label() + ": " + add.Kind.Phrase(), Kind: page.FieldSecret, Required: true})
				answers, err := io.Prompt.Form(page.Form{Title: offer.Intro, Fields: fields})
				if err != nil {
					return err
				}
				if add.Label == "" {
					add.Label = answers["label"]
				}
				add.Value = kit.Typed(answers["value"])
			}
			if add.Label == "" {
				add.Label = program.Label()
			}
			if _, err := call.Invoke(ctx, env.Calls, env.Self, vaultcalls.VaultAdd, add, nil); err != nil {
				return err
			}
			return io.Page(pageaccounts.Added(add.Label, env.Self.Name, d.ThisMachine), nil)
		},
	}, env)
}
