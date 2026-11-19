package wire

import (
	"github.com/spf13/cobra"

	activitycmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/activity"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

func activityCommand(env kit.Env, m machine) *cobra.Command {
	return activitycmd.Command(activitycmd.Deps{ThisMachine: m.where, Now: m.sys.Clock.Now, Label: m.label}, env)
}
