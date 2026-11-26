package wire

import (
	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/project/adapter/manifest"
	"github.com/jterrazz/studio-station/apps/engine/project/contract"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
	projectcmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/project"
	projectcreate "github.com/jterrazz/studio-station/apps/engine/surface/cli/project/create"
	projectdrift "github.com/jterrazz/studio-station/apps/engine/surface/cli/project/drift"
	projectplan "github.com/jterrazz/studio-station/apps/engine/surface/cli/project/plan"
)

func (m machine) projects() contract.Projects {
	if m.world != nil {
		return m.world.projects()
	}
	return manifest.New(m.sys.FS)
}

func projectCommand(env kit.Env, m machine) *cobra.Command {
	folder := local.WorkingDir
	return projectcmd.Command(projectcmd.Deps{ThisMachine: m.where, Folder: folder}, env,
		projectcreate.Command(projectcreate.Deps{ThisMachine: m.where, Folder: folder}, env),
		projectplan.Command(projectplan.Deps{ThisMachine: m.where, Folder: folder}, env),
		projectdrift.Command(projectdrift.Deps{ThisMachine: m.where, Folder: folder}, env),
	)
}
