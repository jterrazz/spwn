package skillscreate

import (
	"context"

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
	Project     func(ctx context.Context, target machine.Ref) (string, error)
}

type Input struct{ Name string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "create <name>",
		Short:   "Make a skill: a markdown file from a template, in this folder's project, or your own outside one",
		Example: "  station skills create paper-reading",
		Args:    cobra.ExactArgs(1),
		Bind:    func(in *Input, args []string) error { in.Name = args[0]; return nil },
		Run: func(ctx context.Context, in *Input, io kit.IO) error {
			project, err := d.Project(ctx, env.Target())
			if err != nil {
				return err
			}
			s, err := call.Invoke(ctx, env.Calls, env.Target(), skillscalls.Create, skillscalls.CreateIn{Project: project, Skill: in.Name}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageskills.Done("skills-create", "skills create", s.Name, "created", s.Path, d.ThisMachine), nil)
		},
	}, env)
}
