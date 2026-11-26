package automationscmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	automationscalls "github.com/jterrazz/studio-station/apps/engine/automations/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageautomations "github.com/jterrazz/studio-station/apps/engine/pages/automations"
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
		Use:     "automations [<automation>]",
		Short:   "The automations this folder's project declares, with what their runs say; one automation's detail",
		Example: "  station automations\n  station automations morning\n  station automations --json",
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
				a, shown := call.Invoke(ctx, env.Calls, env.Target(), automationscalls.Show, automationscalls.ShowIn{Project: project, Automation: in.Name}, nil)
				if shown != nil {
					return shown
				}
				return io.Page(pageautomations.Show(a, d.ThisMachine), pageautomations.ShowContract(automationscalls.Show.Schema(), a))
			}
			all, err := call.Invoke(ctx, env.Calls, env.Target(), automationscalls.List, automationscalls.ListIn{Project: project}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageautomations.List(all, d.ThisMachine), pageautomations.ListContract(automationscalls.List.Schema(), all))
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}
