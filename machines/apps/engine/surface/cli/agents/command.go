package agentscmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	agentscalls "github.com/jterrazz/studio-station/apps/engine/agents/calls"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageagents "github.com/jterrazz/studio-station/apps/engine/pages/agents"
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
		Use:     "agents [<agent>]",
		Short:   "The agents of this folder's project, or your own outside one; one agent's Mind",
		Example: "  station agents\n  station agents neo\n  station agents --json",
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
				m, shown := call.Invoke(ctx, env.Calls, env.Target(), agentscalls.Show, agentscalls.ShowIn{Project: project, Agent: in.Name}, nil)
				if shown != nil {
					return shown
				}
				return io.Page(pageagents.Show(m, d.ThisMachine), pageagents.ShowContract(agentscalls.Show.Schema(), m))
			}
			all, err := call.Invoke(ctx, env.Calls, env.Target(), agentscalls.List, agentscalls.ListIn{Project: project}, nil)
			if err != nil {
				return err
			}
			return io.Page(pageagents.List(all, d.ThisMachine), pageagents.ListContract(agentscalls.List.Schema(), all))
		},
	}, env)
	cmd.AddCommand(children...)
	return cmd
}
