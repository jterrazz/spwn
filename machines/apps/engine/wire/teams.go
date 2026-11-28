package wire

import (
	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/project/adapter/manifest"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
	teamscmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/teams"
	teamsfiles "github.com/jterrazz/studio-station/apps/engine/teams/adapter/files"
	"github.com/jterrazz/studio-station/apps/engine/teams/contract"
)

func (m machine) teams() contract.Teams {
	if m.world != nil {
		return m.world.teams()
	}
	return teamsfiles.Teams{Layouts: manifest.New()}
}

func teamsCommand(env kit.Env, m machine) *cobra.Command {
	return teamscmd.Command(teamscmd.Deps{ThisMachine: m.where, Project: m.agentsProject}, env)
}
