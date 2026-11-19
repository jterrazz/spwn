package backgroundcmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	backgroundcalls "github.com/jterrazz/studio-station/apps/engine/background/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagebackground "github.com/jterrazz/studio-station/apps/engine/pages/background"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct {
	All bool
}

func Command(d Deps, env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Command(kit.Leaf[Input]{
		Use:     "background",
		Short:   "The background tasks of this machine: failing, orphans, station's, the apps', unknown",
		Example: "  station background\n  station background --all\n  station background --json",
		Group:   kit.OnAMachine,
		Args:    cobra.NoArgs,
		Declare: func(f *pflag.FlagSet, in *Input) { f.BoolVar(&in.All, "all", false, "unfold every group") },
		Read:    func(*Input) bool { return true },
		Pages:   true,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			tasks, err := call.Invoke(ctx, env.Calls, env.Target(), backgroundcalls.List, backgroundcalls.ListIn{}, nil)
			if err != nil {
				return err
			}
			return io.Page(pagebackground.List(tasks, d.ThisMachine, in.All), pagebackground.ListContract(tasks))
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}
