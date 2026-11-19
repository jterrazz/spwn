package wire

import (
	"github.com/spf13/cobra"

	backgroundcalls "github.com/jterrazz/studio-station/apps/engine/background/calls"
	"github.com/jterrazz/studio-station/apps/engine/background/contract"
	backgroundcmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/background"
	backgroundremove "github.com/jterrazz/studio-station/apps/engine/surface/cli/background/remove"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

func backgroundCommand(env kit.Env, m machine) *cobra.Command {
	return backgroundcmd.Command(backgroundcmd.Deps{ThisMachine: m.where}, env,
		backgroundremove.Command(backgroundremove.Deps{ThisMachine: m.where}, env))
}

func (m machine) backgroundDeps() backgroundcalls.Deps {
	return backgroundcalls.Deps{
		Jobs: m.jobs(0), Domain: contract.Domain{UID: m.sys.Env.Euid()},
		Home: m.sys.Env.Home(), Root: m.systemRoot(), Kind: m.profile.Kind,
	}
}
