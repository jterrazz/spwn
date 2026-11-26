package devicesforget

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	devicescalls "github.com/jterrazz/studio-station/apps/engine/devices/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagedevices "github.com/jterrazz/studio-station/apps/engine/pages/devices"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct{ Name, Confirm string }

func Command(d Deps, env kit.Env) *cobra.Command {
	// invariant: forget asks the device's name typed, never yes: on a
	// terminal the question, in a script --confirm=<name>.
	return kit.Command(kit.Leaf[Input]{
		Use:     "forget <name>",
		Short:   "Take a device out of this Mac's device book: no new login into this Mac's VMs, but one into this Mac still reaches them",
		Example: "  station devices forget macbook-pro --confirm=macbook-pro\n  station devices forget phone --on mac-mini --confirm=phone",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Confirm, "confirm", "", "the device's name, typed, to forget it without the question")
		},
		Bind: func(in *Input, args []string) error {
			in.Name = args[0]
			return nil
		},
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			forget := func(answer string) (devicescalls.Forgotten, error) {
				return call.Invoke(ctx, env.Calls, env.Target(), devicescalls.Forget, devicescalls.ForgetIn{Name: in.Name, Confirm: answer}, nil)
			}
			out, err := forget(in.Confirm)
			if _, typed := confirm.Answer(err); typed && in.Confirm == "" && io.Interactive && io.Prompt != nil {
				answers, askErr := io.Prompt.Form(page.Form{Fields: []page.Field{{Key: "name", Label: "Type " + in.Name + " to forget it:", Kind: page.FieldText}}})
				if askErr != nil {
					return askErr
				}
				out, err = forget(answers["name"])
				var wrong *fail.Error
				if errors.As(err, &wrong) && wrong.Code == confirm.Mismatch {
					return fail.New(fail.Declined, confirm.Mismatch, wrong.Subject, wrong.Detail)
				}
			}
			if err != nil {
				return err
			}
			return io.Page(pagedevices.Forgotten(out.Name, env.Target().Name, d.ThisMachine), out)
		},
	}, env)
}
