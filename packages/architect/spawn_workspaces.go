package architect

import (
	"fmt"
	"path/filepath"

	"spwn.sh/packages/platform"
	"spwn.sh/packages/world/models"
)

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

func workspaceContainerPath(name string, totalWorkspaces int) string {
	_ = totalWorkspaces
	return "/workspaces/" + name
}

func worldStateDirFor(worldID string) string {
	return filepath.Join(platform.LocalStateDir(), "world-states", worldID)
}
