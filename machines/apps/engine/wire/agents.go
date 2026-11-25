package wire

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/agents/adapter/mind"
	agentscalls "github.com/jterrazz/studio-station/apps/engine/agents/calls"
	"github.com/jterrazz/studio-station/apps/engine/agents/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	agentscmd "github.com/jterrazz/studio-station/apps/engine/surface/cli/agents"
	agentscreate "github.com/jterrazz/studio-station/apps/engine/surface/cli/agents/create"
	agentsdelete "github.com/jterrazz/studio-station/apps/engine/surface/cli/agents/delete"
	agentsdream "github.com/jterrazz/studio-station/apps/engine/surface/cli/agents/dream"
	agentsedit "github.com/jterrazz/studio-station/apps/engine/surface/cli/agents/edit"
	agentsexport "github.com/jterrazz/studio-station/apps/engine/surface/cli/agents/export"
	agentsfork "github.com/jterrazz/studio-station/apps/engine/surface/cli/agents/fork"
	agentsimport "github.com/jterrazz/studio-station/apps/engine/surface/cli/agents/import"
	agentssleep "github.com/jterrazz/studio-station/apps/engine/surface/cli/agents/sleep"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
)

// value: the file whose folder is a project's root.
const projectManifest = "spwn.yaml"

func (m machine) minds() contract.Minds {
	if m.world != nil {
		return m.world.minds()
	}
	return mind.Minds{RemoveAll: m.sys.FS.RemoveAll}
}

func agentsCommand(env kit.Env, m machine) *cobra.Command {
	project := m.agentsProject
	return agentscmd.Command(agentscmd.Deps{ThisMachine: m.where, Project: project}, env,
		agentscreate.Command(agentscreate.Deps{ThisMachine: m.where, Project: project}, env),
		agentsedit.Command(agentsedit.Deps{ThisMachine: m.where, Project: project}, env),
		agentsdelete.Command(agentsdelete.Deps{ThisMachine: m.where, Project: project}, env),
		agentsfork.Command(agentsfork.Deps{ThisMachine: m.where, Project: project}, env),
		agentsexport.Command(agentsexport.Deps{ThisMachine: m.where, Project: project}, env),
		agentsimport.Command(agentsimport.Deps{ThisMachine: m.where, Project: project}, env),
		agentsdream.Command(agentsdream.Deps{ThisMachine: m.where, Project: project}, env),
		agentssleep.Command(agentssleep.Deps{ThisMachine: m.where, Project: project}, env),
	)
}

// invariant: the project of the folder a command runs in, found by its
// manifest from that folder up; on this machine its root as it is, on a VM
// of this Mac the guest folder a share of its start puts it at. A project
// no share reaches, or a target that is not this Mac's own VM, is refused:
// the target would act on other agents than this folder's.
func (m machine) agentsProject(ctx context.Context, target coremachine.Ref) (string, error) {
	root, err := m.projectRoot()
	if err != nil || root == "" {
		return "", err
	}
	if target.Name == "" || target.Name == m.profile.Ref.Name {
		return root, nil
	}
	refused := func(vm bool) error {
		detail := map[string]string{"path": m.homeWord(root)}
		if vm {
			detail["vm"] = "true"
		}
		return fail.New(fail.Refused, agentscalls.NotShared, target.Name, detail)
	}
	record, err := m.stores.machines.Get(ctx, target.Name)
	if err != nil {
		return "", refused(false)
	}
	for _, share := range record.Mounts {
		if rest, ok := within(root, share.Computer); ok {
			return strings.TrimSuffix(share.Guest, "/") + rest, nil
		}
	}
	return "", refused(true)
}

func within(path, folder string) (string, bool) {
	folder = strings.TrimSuffix(folder, "/")
	if path == folder {
		return "", true
	}
	if rest, ok := strings.CutPrefix(path, folder+"/"); ok {
		return "/" + rest, true
	}
	return "", false
}

func (m machine) projectRoot() (string, error) {
	dir := local.WorkingDir()
	if dir == "" {
		return "", nil
	}
	for {
		if info, err := m.sys.FS.Stat(filepath.Join(dir, projectManifest)); err == nil && !info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

func (m machine) homeWord(path string) string {
	if rest, ok := within(path, m.sys.Env.Home()); ok {
		return "~" + rest
	}
	return path
}
