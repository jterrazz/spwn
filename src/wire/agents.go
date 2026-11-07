package wire

import (
	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/page"
	agentscmd "github.com/jterrazz/studio-station/src/surface/cli/agents"
	agentsmergepr "github.com/jterrazz/studio-station/src/surface/cli/agents/mergepr"
	agentsrecord "github.com/jterrazz/studio-station/src/surface/cli/agents/record"
	agentsreply "github.com/jterrazz/studio-station/src/surface/cli/agents/reply"
	agentsstart "github.com/jterrazz/studio-station/src/surface/cli/agents/start"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

func agentsCommand(env kit.Env, where page.Context) *cobra.Command {
	return agentscmd.Command(agentscmd.Deps{ThisMachine: where}, env,
		agentsrecord.Command(env),
		agentsreply.Command(agentsreply.Deps{ThisMachine: where}, env),
		agentsstart.Command(agentsstart.Deps{ThisMachine: where}, env),
		agentsmergepr.Command(agentsmergepr.Deps{ThisMachine: where}, env))
}
