package architect

import (
	"fmt"
	"path/filepath"

	"spwn.sh/packages/transpile/worldbook"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/world/models"
)

// buildWorkspaceBinds generates Docker bind specs for the resolved
// workspaces. Layout is uniform:
//
//   - 0 workspaces: no binds. /workspaces does not exist; the
//     agent's only writable space is its own home at /agents/<name>.
//   - 1+ workspaces: each mounted at /workspaces/<name>. There is no
//     special-cased single-workspace path — `ls /workspaces` always
//     tells the agent what projects it can touch.
func buildWorkspaceBinds(workspaces []models.Workspace) []string {
	if len(workspaces) == 0 {
		return nil
	}
	binds := make([]string, 0, len(workspaces))
	for _, ws := range workspaces {
		ro := ""
		if ws.ReadOnly {
			ro = ":ro"
		}
		binds = append(binds, fmt.Sprintf("%s:/workspaces/%s%s", ws.Path, ws.Name, ro))
	}
	return binds
}

// workspaceContainerPath returns the absolute path inside the container
// where a workspace named `name` is mounted. Single source of truth
// for the container-side workspace path scheme.
func workspaceContainerPath(name string, totalWorkspaces int) string {
	_ = totalWorkspaces // legacy parameter; layout is uniform now
	return "/workspaces/" + name
}

// worldStateDirFor returns the host-side directory where a given
// world's per-instance state is stored. Used by both spawn (initial
// write) and DeployAgent (roster regeneration).
func worldStateDirFor(worldID string) string {
	return filepath.Join(platform.LocalStateDir(), "world-states", worldID)
}

// convertWorkspaces adapts the world-layer Workspace type to the
// compile-layer Workspace type for rendering.
func convertWorkspaces(ws []models.Workspace) []worldbook.Workspace {
	out := make([]worldbook.Workspace, len(ws))
	for i, w := range ws {
		out[i] = worldbook.Workspace{Name: w.Name, Path: w.Path, ReadOnly: w.ReadOnly}
	}
	return out
}
