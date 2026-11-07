package agentscmd

import (
	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/surface/cli/kit"
)

func Command(env kit.Env, children ...*cobra.Command) *cobra.Command {
	cmd := kit.Group("agents", "The agents working in this workspace", "", env, children...)
	cmd.Hidden = true
	return cmd
}
