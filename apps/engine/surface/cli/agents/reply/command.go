package agentsreply

import (
	"context"
	"io"
	"strings"

	"github.com/spf13/cobra"

	agentscalls "github.com/jterrazz/studio-station/apps/engine/agents/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageagents "github.com/jterrazz/studio-station/apps/engine/pages/agents"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
	// value: the workspace the calls for the agent of that name go to.
	Where func(ctx context.Context, name string) (machine.Ref, error)
}

type Input struct {
	Agent string
	Text  []string
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "reply <agent> [text]",
		Short:   "Type a reply to a waiting agent: it works on it",
		Example: "  station agents reply review-bot yes, merge it\n  echo 'yes, merge it' | station agents reply review-bot",
		Args:    cobra.MinimumNArgs(1),
		Bind: func(in *Input, args []string) error {
			in.Agent, in.Text = args[0], args[1:]
			return nil
		},
		Run: func(ctx context.Context, in *Input, o kit.IO) error {
			text := strings.Join(in.Text, " ")
			if text == "" && !env.Interactive && env.Stdin != nil {
				data, _ := io.ReadAll(io.LimitReader(env.Stdin, catalog.AgentPayloadCap))
				text = strings.TrimRight(string(data), "\n")
			}
			at, err := d.Where(ctx, in.Agent)
			if err != nil {
				return err
			}
			a, err := call.Invoke(ctx, env.Calls, at, agentscalls.Reply, agentscalls.ReplyIn{Agent: in.Agent, Text: text}, nil)
			if err != nil {
				return err
			}
			return o.Page(pageagents.Replied(a, d.ThisMachine), a)
		},
	}, env)
}
