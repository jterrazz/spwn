package settingsoff

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagesettings "github.com/jterrazz/studio-station/apps/engine/pages/settings"
	"github.com/jterrazz/studio-station/apps/engine/settings"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	Names       func() []string
	ThisMachine page.Context
	// value: what else turning these settings off does on the target, said
	// in the question.
	Also func(names []string) []string
}

type Input struct{ Names []string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "off <setting...>",
		Short:   "Turn settings off here: station neither applies nor watches them; a change of yours is kept",
		Example: "  station settings off starship-dotfiles --yes",
		Args:    cobra.MinimumNArgs(1),
		Bind:    func(in *Input, args []string) error { in.Names = args; return nil },
		Asks:    true,
		Confirm: func(in *Input) string {
			var also []string
			if d.Also != nil {
				also = d.Also(in.Names)
			}
			return pagesettings.OffConfirm(in.Names, also...).Question
		},
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			var changes []settings.Change
			err := io.Confirmed(func(answer string) (err error) {
				changes, err = call.Invoke(ctx, env.Calls, env.Target(), settingscalls.Off, settingscalls.OffIn{Names: in.Names, Confirm: answer}, nil)
				return err
			})
			if err != nil {
				return err
			}
			return io.Page(pagesettings.Changes("off", changes, d.ThisMachine), nil)
		},
	}, env)
}
