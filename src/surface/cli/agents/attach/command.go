package agentsattach

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/agents"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	// value: puts this terminal on the terminal at that address of machine,
	// until the owner detaches; an empty machine is this one.
	Attach func(ctx context.Context, machine, terminal string) error
	// value: the agent of that name, and the workspace it runs on.
	Find func(ctx context.Context, name string) (agents.Agent, machine.Ref, error)
}

type Input struct{ Agent string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "attach <agent>",
		Short:   "Put this terminal on an agent's own terminal, until you detach",
		Example: "  station agents attach review-bot",
		Args:    cobra.ExactArgs(1),
		Bind: func(in *Input, args []string) error {
			in.Agent = args[0]
			return nil
		},
		Run: func(ctx context.Context, in *Input, _ kit.IO) error {
			// invariant: the agent is found where it runs, and the owner's
			// terminal is put on its terminal from here, through this
			// machine's multiplexer.
			a, at, err := d.Find(ctx, in.Agent)
			if err != nil {
				return err
			}
			if a.Terminal == "" || !agents.Live(a) {
				return fail.New(fail.Refused, agentscalls.NoTerminal, in.Agent, nil)
			}
			on := ""
			if at != env.Self {
				on = at.Name
			}
			return d.Attach(ctx, on, a.Terminal)
		},
	}, env)
}
