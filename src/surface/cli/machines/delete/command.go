package machinesdelete

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/confirm"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/machines"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	"github.com/jterrazz/studio-station/src/page"
	pagemachines "github.com/jterrazz/studio-station/src/pages/machines"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	// value: the machines known here, for completion.
	Names       func() []string
	ThisMachine page.Context
}

type Input struct{ Name, Confirm string }

func Command(d Deps, env kit.Env) *cobra.Command {
	// invariant: a delete asks the workspace's name typed, never yes: on a
	// terminal the question, in a script --confirm=<name>; --yes answers nothing.
	return kit.Command(kit.Leaf[Input]{
		Use:     "delete <vm>",
		Short:   "Delete a workspace: its disk, its record, its reserved ports and its shared accounts",
		Example: "  station machines delete linux-1 --confirm=linux-1",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Confirm, "confirm", "", "the workspace's name, typed, to delete it without the question")
		},
		Bind: func(in *Input, args []string) error {
			in.Name = args[0]
			return nil
		},
		Complete: func([]string, string) []string { return d.Names() },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			deleteIt := func(answer string) (machines.DeleteReport, error) {
				report, err := call.Invoke(ctx, env.Calls, env.Target(), machinescalls.Delete, machinescalls.DeleteIn{VM: in.Name, Confirm: answer}, nil)
				return report, kit.OnItsMac(ctx, env, in.Name, err)
			}
			report, err := deleteIt(in.Confirm)
			if _, typed := confirm.Answer(err); typed && in.Confirm == "" && io.Interactive && io.Prompt != nil {
				label := "Type " + in.Name + " to delete it:"
				var asked *fail.Error
				if errors.As(err, &asked) && asked.Detail["agents"] != "" {
					label = "Its agents stop first: " + asked.Detail["agents"] + ". " + label
				}
				answers, askErr := io.Prompt.Form(page.Form{Fields: []page.Field{{Key: "name", Label: label, Kind: page.FieldText}}})
				if askErr != nil {
					return askErr
				}
				stop := io.Wait("deleting " + in.Name)
				report, err = deleteIt(answers["name"])
				stop()
				var wrong *fail.Error
				if errors.As(err, &wrong) && wrong.Code == confirm.Mismatch {
					return fail.New(fail.Declined, confirm.Mismatch, wrong.Subject, wrong.Detail)
				}
			}
			if err != nil {
				return err
			}
			if env.Use != nil {
				env.Use.Gone(in.Name, env.Use.Now())
			}
			return io.Page(pagemachines.Deleted(report, d.ThisMachine), nil)
		},
	}, env)
}
