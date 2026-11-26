package projectcmd

import (
	"context"
	"errors"

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
	// value: the folder the command runs in, an absolute path.
	Folder func() string
}

type Input struct{}

func Command(d Deps, env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Command(kit.Leaf[Input]{
		Use:     "project",
		Short:   "The project of this folder: its file and its agents",
		Example: "  station project\n  station project --json",
		Group:   kit.Agent,
		Args: func(c *cobra.Command, args []string) error {
			if len(args) > 0 {
				return errors.New("unknown command \"" + args[0] + "\" for \"" + c.CommandPath() + "\"")
			}
			return nil
		},
		Read:  func(*Input) bool { return true },
		Pages: true,
		Run: func(ctx context.Context, _ *Input, io kit.IO) error {
			p, err := call.Invoke(ctx, env.Calls, machine.Ref{}, projectcalls.Show, projectcalls.ShowIn{Folder: d.Folder()}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageproject.Show(p, d.ThisMachine), pageproject.ShowContract(projectcalls.Show.Schema(), p))
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}
