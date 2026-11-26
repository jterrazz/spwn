package teamscmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageteams "github.com/jterrazz/studio-station/apps/engine/pages/teams"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
	teamscalls "github.com/jterrazz/studio-station/apps/engine/teams/calls"
)

type Deps struct {
	ThisMachine page.Context
	// value: the root of this folder's project as the target reads it; empty
	// for none, refused when the target cannot read it.
	Project func(ctx context.Context, target machine.Ref) (string, error)
}

type Input struct{ Slug string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "teams [<team>]",
		Short:   "The teams declared for you, with the agents of this folder's project in each; one team's detail",
		Example: "  station teams\n  station teams web\n  station teams --json",
		Group:   kit.Agent,
		Args: func(c *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("unknown command \"" + args[0] + "\" for \"" + c.CommandPath() + "\"")
			}
			return nil
		},
		Bind: func(in *Input, args []string) error {
			if len(args) == 1 {
				in.Slug = args[0]
			}
			return nil
		},
		Read:  func(*Input) bool { return true },
		Pages: true,
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			project, err := d.Project(ctx, env.Target())
			if err != nil {
				return err
			}
			if in.Slug != "" {
				t, shown := call.Invoke(ctx, env.Calls, env.Target(), teamscalls.Show, teamscalls.ShowIn{Project: project, Team: in.Slug}, nil)
				if shown != nil {
					return shown
				}
				return io.Page(pageteams.Show(t, d.ThisMachine), pageteams.ShowContract(teamscalls.Show.Schema(), t))
			}
			all, err := call.Invoke(ctx, env.Calls, env.Target(), teamscalls.List, teamscalls.ListIn{Project: project}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageteams.List(all, d.ThisMachine), pageteams.ListContract(teamscalls.List.Schema(), all))
		},
	}, env)
}
