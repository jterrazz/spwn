package wire

import (
	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/skills/adapter/files"
	"github.com/jterrazz/studio-station/apps/engine/skills/contract"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
	skillscmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/skills"
	skillscreate "github.com/jterrazz/studio-station/apps/engine/surface/cli/skills/create"
	skillsdelete "github.com/jterrazz/studio-station/apps/engine/surface/cli/skills/delete"
)

func (m machine) skillFiles() contract.Files {
	if m.world != nil {
		return m.world.skillFiles()
	}
	return files.New(m.sys.FS)
}

func skillsCommand(env kit.Env, m machine) *cobra.Command {
	project := m.agentsProject
	return skillscmd.Command(skillscmd.Deps{ThisMachine: m.where, Project: project}, env,
		skillscreate.Command(skillscreate.Deps{ThisMachine: m.where, Project: project}, env),
		skillsdelete.Command(skillsdelete.Deps{ThisMachine: m.where, Project: project}, env),
	)
}
