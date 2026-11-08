package agentsattach

import (
	"context"
	"slices"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/agents"
	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	// value: puts this terminal on the terminal at that address of machine,
	// until the owner detaches; an empty machine is this one.
	Attach func(ctx context.Context, machine, terminal string) error
}

type Input struct{ Session string }

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "attach <session>",
		Short:   "Put this terminal on an agent's, until you detach",
		Example: "  station agents attach s-1 --on linux-1",
		Args:    cobra.ExactArgs(1),
		Bind: func(in *Input, args []string) error {
			in.Session = args[0]
			return nil
		},
		Run: func(ctx context.Context, in *Input, _ kit.IO) error {
			// invariant: the agent is found where it runs, and the owner's
			// terminal is put on its terminal from here, through this
			// machine's multiplexer.
			kept, err := call.Invoke(ctx, env.Calls, env.Target(), agentscalls.List, struct{}{}, nil)
			if err != nil {
				return err
			}
			i := slices.IndexFunc(kept, func(a agents.Agent) bool { return a.Session == in.Session })
			if i < 0 {
				return fail.New(fail.NotFound, agentscalls.UnknownSession, in.Session, nil)
			}
			if kept[i].Terminal == "" {
				return fail.New(fail.Refused, agentscalls.NoTerminal, in.Session, nil)
			}
			machine := ""
			if env.Far() {
				machine = env.Target().Name
			}
			return d.Attach(ctx, machine, kept[i].Terminal)
		},
	}, env)
}
