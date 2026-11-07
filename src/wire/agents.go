package wire

import (
	"github.com/spf13/cobra"

	agentscmd "github.com/jterrazz/studio-station/src/surface/cli/agents"
	agentsrecord "github.com/jterrazz/studio-station/src/surface/cli/agents/record"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

func agentsCommand(env kit.Env) *cobra.Command {
	return agentscmd.Command(env, agentsrecord.Command(env))
}
