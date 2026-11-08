package agentsstop

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/agents"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/page"
	pageagents "github.com/jterrazz/studio-station/src/pages/agents"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	// value: the workspace the calls for the agent of that name go to, and
	// the name it knows the agent by.
	Where func(ctx context.Context, name string) (machine.Ref, string, error)
}

type Input struct{ Agent string }

func Command(d Deps, env kit.Env) *cobra.Command {
	// invariant: SPEC B5.22: a stop asks yes or no (--yes), and never
	// touches the agent's workspace.
	return kit.Command(kit.Leaf[Input]{
		Use:     "stop <agent>",
		Short:   "Stop an agent: its terminal closes, its workspace keeps running",
		Example: "  station agents stop review-bot --yes",
		Args:    cobra.ExactArgs(1),
		Bind:    func(in *Input, args []string) error { in.Agent = args[0]; return nil },
		Asks:    true,
		Confirm: func(in *Input) string {
			return "Stop the agent " + in.Agent + "? Its terminal closes; its workspace keeps running."
		},
		Verb: func(*Input) string { return "stop" },
		Run: func(ctx context.Context, in *Input, o kit.IO) error {
			at, local, err := d.Where(ctx, in.Agent)
			if err != nil {
				return err
			}
			var a agents.Agent
			err = o.Confirmed(func(answer string) (err error) {
				a, err = call.Invoke(ctx, env.Calls, at, agentscalls.Stop, agentscalls.StopIn{Agent: local, Confirm: answer}, nil)
				return err
			})
			if err != nil {
				return err
			}
			a.Name = in.Agent
			return o.Page(pageagents.Stopped(a, d.ThisMachine), a)
		},
	}, env)
}
