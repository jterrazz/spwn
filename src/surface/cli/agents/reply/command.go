package agentsreply

import (
	"context"
	"io"
	"strings"

	"github.com/spf13/cobra"

	agentscalls "github.com/jterrazz/studio-station/src/agents/calls"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/page"
	pageagents "github.com/jterrazz/studio-station/src/pages/agents"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type Deps struct {
	ThisMachine page.Context
}

type Input struct {
	Session string
	Text    []string
}

func Command(d Deps, env kit.Env) *cobra.Command {
	return kit.Command(kit.Leaf[Input]{
		Use:     "reply <session> [text]",
		Short:   "Type a reply to a waiting agent: it works on it",
		Example: "  station agents reply s-1 yes, merge it\n  echo 'yes, merge it' | station agents reply s-1 --on linux-1",
		Args:    cobra.MinimumNArgs(1),
		Bind: func(in *Input, args []string) error {
			in.Session, in.Text = args[0], args[1:]
			return nil
		},
		Run: func(ctx context.Context, in *Input, o kit.IO) error {
			text := strings.Join(in.Text, " ")
			if text == "" && !env.Interactive && env.Stdin != nil {
				data, _ := io.ReadAll(io.LimitReader(env.Stdin, catalog.AgentPayloadCap))
				text = strings.TrimRight(string(data), "\n")
			}
			a, err := call.Invoke(ctx, env.Calls, env.Target(), agentscalls.Reply, agentscalls.ReplyIn{Session: in.Session, Text: text}, nil)
			if err != nil {
				return err
			}
			return o.Page(pageagents.Replied(a, d.ThisMachine), a)
		},
	}, env)
}
