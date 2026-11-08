package wire

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/page"
	agentscmd "github.com/jterrazz/studio-station/src/surface/cli/agents"
	agentsattach "github.com/jterrazz/studio-station/src/surface/cli/agents/attach"
	agentsmergepr "github.com/jterrazz/studio-station/src/surface/cli/agents/mergepr"
	agentsrecord "github.com/jterrazz/studio-station/src/surface/cli/agents/record"
	agentsreply "github.com/jterrazz/studio-station/src/surface/cli/agents/reply"
	agentsstart "github.com/jterrazz/studio-station/src/surface/cli/agents/start"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

type agentsDeps struct {
	// value: the terminal of the multiplexer this process runs in, empty outside one.
	Terminal string
	Attach   func(ctx context.Context, machine, terminal string) error
}

func agentsCommand(env kit.Env, where page.Context, d agentsDeps) *cobra.Command {
	return agentscmd.Command(agentscmd.Deps{ThisMachine: where}, env,
		agentsrecord.Command(agentsrecord.Deps{Terminal: d.Terminal}, env),
		agentsattach.Command(agentsattach.Deps{Attach: d.Attach}, env),
		agentsreply.Command(agentsreply.Deps{ThisMachine: where}, env),
		agentsstart.Command(agentsstart.Deps{ThisMachine: where}, env),
		agentsmergepr.Command(agentsmergepr.Deps{ThisMachine: where}, env))
}
