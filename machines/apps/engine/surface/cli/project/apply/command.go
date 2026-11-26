package projectapply

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageproject "github.com/jterrazz/studio-station/apps/engine/pages/project"
	"github.com/jterrazz/studio-station/apps/engine/project"
	projectcalls "github.com/jterrazz/studio-station/apps/engine/project/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	Folder      func() string
}

type Input struct {
	// value: what the apply would lose, once its call asked, and its
	// refusal, which names each file and --yes.
	asked   project.Applied
	refused error
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "apply",
		Short:   "Render each agent into its run folder, as the project declares it; asks before it overwrites or removes a file",
		Example: "  station project apply\n  station project apply --yes\n  station project apply --json --yes",
		Read:    func(*Input) bool { return true },
		Pages:   true,
		Asks:    true,
		Ask: func(in *Input) (bool, error) {
			return env.Prompt.Confirm(pageproject.ApplyConfirm(in.asked))
		},
		Refuse: func(in *Input) error { return in.refused },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			// invariant: the plan is read first: one with an error is
			// printed and refused, and on a page what apply changes is
			// listed before anything is written.
			p, err := call.Invoke(ctx, env.Calls, machine.Ref{}, projectcalls.Plan, projectcalls.PlanIn{Folder: d.Folder()}, nil)
			if err != nil {
				return err
			}
			if !p.Valid {
				if printed := io.Page(pageproject.Plan(p, d.ThisMachine), pageproject.PlanContract(projectcalls.Plan.Schema(), p)); printed != nil {
					return printed
				}
				return projectcalls.Refused(p)
			}
			if len(p.Changes) > 0 && !io.JSON && !io.PageJSON {
				if printed := io.Page(pageproject.Pending(p, d.ThisMachine), nil); printed != nil {
					return printed
				}
			}
			var out project.Applied
			err = io.Confirmed(func(answer string) (err error) {
				out, err = call.Invoke(ctx, env.Calls, machine.Ref{}, projectcalls.Apply, projectcalls.ApplyIn{Folder: d.Folder(), Confirm: answer}, nil)
				if _, asks := confirm.Answer(err); asks {
					in.asked, in.refused = out, err
				}
				return err
			})
			if err != nil {
				return err
			}
			return io.Page(pageproject.Applied(out, d.ThisMachine), pageproject.AppliedContract(projectcalls.Apply.Schema(), out))
		},
	}, env)
}
