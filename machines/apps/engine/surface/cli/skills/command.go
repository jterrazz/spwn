package skillscmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageskills "github.com/jterrazz/studio-station/apps/engine/pages/skills"
	skillscalls "github.com/jterrazz/studio-station/apps/engine/skills/calls"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	// value: the root of this folder's project as the target reads it; empty
	// for none, refused when the target cannot read it.
	Project func(ctx context.Context, target machine.Ref) (string, error)
}

type Input struct{ Name string }

func Command(d Deps, env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Command(kit.Leaf[Input]{
		Use:     "skills [<skill>]",
		Short:   "The skill files of this folder's project, or your own outside one; one skill's text",
		Example: "  station skills\n  station skills paper-reading\n  station skills --json",
		Group:   kit.Agent,
		Args: func(c *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("unknown command \"" + args[0] + "\" for \"" + c.CommandPath() + "\"")
			}
			return nil
		},
		Bind: func(in *Input, args []string) error {
			if len(args) == 1 {
				in.Name = args[0]
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
			if in.Name != "" {
				s, shown := call.Invoke(ctx, env.Calls, env.Target(), skillscalls.Show, skillscalls.ShowIn{Project: project, Skill: in.Name}, nil)
				if shown != nil {
					return shown
				}
				return io.Page(pageskills.Show(s, d.ThisMachine), pageskills.ShowContract(skillscalls.Show.Schema(), s))
			}
			all, err := call.Invoke(ctx, env.Calls, env.Target(), skillscalls.List, skillscalls.ListIn{Project: project}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageskills.List(all, d.ThisMachine), pageskills.ListContract(skillscalls.List.Schema(), all))
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}
