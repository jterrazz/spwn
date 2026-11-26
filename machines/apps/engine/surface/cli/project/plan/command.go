package projectplan

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageproject "github.com/jterrazz/studio-station/apps/engine/pages/project"
	projectcalls "github.com/jterrazz/studio-station/apps/engine/project/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	Folder      func() string
}

type Input struct{}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "plan",
		Short:   "What the project's rules find in its files, the errors first; a plan with an error exits 2",
		Example: "  station project plan\n  station project plan --json",
		Read:    func(*Input) bool { return true },
		Pages:   true,
		Run: func(ctx context.Context, _ *Input, io kit.IO) error {
			p, err := call.Invoke(ctx, env.Calls, machine.Ref{}, projectcalls.Plan, projectcalls.PlanIn{Folder: d.Folder()}, nil)
			if err != nil {
				return err
			}
			if err := io.Page(pageproject.Plan(p, d.ThisMachine), pageproject.PlanContract(projectcalls.Plan.Schema(), p)); err != nil {
				return err
			}
			return projectcalls.Refused(p)
		},
	}, env)
}
