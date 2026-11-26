package projectdrift

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
		Use:     "drift",
		Short:   "How what was installed for the project differs from what it declares; a row refuses nothing",
		Example: "  station project drift\n  station project drift --json",
		Read:    func(*Input) bool { return true },
		Pages:   true,
		Run: func(ctx context.Context, _ *Input, io kit.IO) error {
			d2, err := call.Invoke(ctx, env.Calls, machine.Ref{}, projectcalls.Drift, projectcalls.DriftIn{Folder: d.Folder()}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageproject.Drift(d2, d.ThisMachine), pageproject.DriftContract(projectcalls.Drift.Schema(), d2))
		},
	}, env)
}
