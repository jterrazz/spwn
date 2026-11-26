package skillsdelete

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageskills "github.com/jterrazz/studio-station/apps/engine/pages/skills"
	skillscalls "github.com/jterrazz/studio-station/apps/engine/skills/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	Project     func(ctx context.Context, target machine.Ref) (string, error)
}

type Input struct{ Name, Confirm string }

func Command(d Deps, env kit.Env) *cobra.Command {
	// invariant: a delete asks the skill's name typed, never yes: on a
	// terminal the question, in a script --confirm=<name>.
	return kit.Command(kit.Leaf[Input]{
		Use:     "delete <skill>",
		Short:   "Delete a skill: its file, for good",
		Example: "  station skills delete paper-reading --confirm=paper-reading",
		Args:    cobra.ExactArgs(1),
		Declare: func(f *pflag.FlagSet, in *Input) {
			f.StringVar(&in.Confirm, "confirm", "", "the skill's name, typed, to delete it without the question")
		},
		Bind: func(in *Input, args []string) error { in.Name = args[0]; return nil },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			project, err := d.Project(ctx, env.Target())
			if err != nil {
				return err
			}
			deleteIt := func(answer string) error {
				_, deleted := call.Invoke(ctx, env.Calls, env.Target(), skillscalls.Delete, skillscalls.DeleteIn{Project: project, Skill: in.Name, Confirm: answer}, nil)
				return deleted
			}
			err = deleteIt(in.Confirm)
			if _, typed := confirm.Answer(err); typed && in.Confirm == "" && io.Interactive && io.Prompt != nil {
				answers, askErr := io.Prompt.Form(page.Form{Fields: []page.Field{{Key: "name", Label: "Type " + in.Name + " to delete it:", Kind: page.FieldText}}})
				if askErr != nil {
					return askErr
				}
				err = deleteIt(answers["name"])
				var wrong *fail.Error
				if errors.As(err, &wrong) && wrong.Code == confirm.Mismatch {
					return fail.New(fail.Declined, confirm.Mismatch, wrong.Subject, wrong.Detail)
				}
			}
			if err != nil {
				return err
			}
			return io.Page(pageskills.Done("skills-delete", "skills delete", in.Name, "deleted", "", d.ThisMachine), nil)
		},
	}, env)
}
