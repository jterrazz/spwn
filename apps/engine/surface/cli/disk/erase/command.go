package diskerase

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/disk"
	diskcalls "github.com/jterrazz/studio-station/apps/engine/disk/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagedisk "github.com/jterrazz/studio-station/apps/engine/pages/disk"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	Names       func() []string
	ThisMachine page.Context
}

type Input struct{ App, Confirm string }

func Command(d Deps, env kit.Env) *cobra.Command {
	// invariant: an erase asks the app's name typed, never yes: on a terminal
	// the question, in a script --confirm=<app>; --yes answers nothing.
	return kit.Command(kit.Leaf[Input]{
		Use:     "erase <app>",
		Short:   "Erase an app's data, back to factory; asks its name typed",
		Example: "  station disk erase xcode\n  station disk erase xcode --confirm=xcode",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Confirm, "confirm", "", "the app's name, typed, to erase it without the question")
		},
		Bind:     func(in *Input, args []string) error { in.App = args[0]; return nil },
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			eraseIt := func(answer string) (disk.Outcome, error) {
				return call.Invoke(ctx, env.Calls, env.Target(), diskcalls.Erase, diskcalls.EraseIn{App: in.App, Confirm: answer}, nil)
			}
			outcome, err := eraseIt(in.Confirm)
			if _, typed := confirm.Answer(err); typed && in.Confirm == "" && io.Interactive && io.Prompt != nil {
				answers, askErr := io.Prompt.Form(page.Form{Fields: []page.Field{{Key: "name", Label: "Type " + in.App + " to erase its data:", Kind: page.FieldText}}})
				if askErr != nil {
					return askErr
				}
				stop := io.Wait("erasing " + in.App)
				outcome, err = eraseIt(answers["name"])
				stop()
				var wrong *fail.Error
				if errors.As(err, &wrong) && wrong.Code == confirm.Mismatch {
					declined := fail.New(fail.Declined, confirm.Mismatch, wrong.Subject, wrong.Detail)
					declined.Message = "not erased: you typed \"" + wrong.Detail["typed"] + "\", not \"" + wrong.Subject + "\""
					declined.Hint = "station disk erase " + wrong.Subject
					return declined
				}
			}
			if err != nil {
				return err
			}
			if err := io.Page(pagedisk.Erased(outcome, d.ThisMachine), nil); err != nil {
				return err
			}
			if failed, _ := disk.Failures([]disk.Outcome{outcome}); failed > 0 {
				return kit.PartialFailure{Failed: 1, Total: 1, Verb: "could not be erased", FirstHint: "station disk erase " + in.App}
			}
			return nil
		},
	}, env)
}
