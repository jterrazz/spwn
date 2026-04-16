package migrations

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
)

// WorkspaceToWorkspaces converts the legacy single "workspace" string field
// to a "workspaces" array in state.json.
var WorkspaceToWorkspaces = migration004()

func migration004() Migration {
	return Migration{
		Number:      4,
		Description: "convert workspace string to workspaces array in state.json",
		Apply: func(_ context.Context, baseDir string) error {
			path := filepath.Join(baseDir, "state.json")
			data, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if len(data) == 0 {
				return nil
			}

			var worlds []map[string]any
			if err := json.Unmarshal(data, &worlds); err != nil {
				return err
			}

			changed := false
			for i, w := range worlds {
				ws, hasWorkspace := w["workspace"]
				_, hasWorkspaces := w["workspaces"]
				if !hasWorkspace || hasWorkspaces {
					continue
				}
				wsStr, ok := ws.(string)
				if !ok || wsStr == "" {
					continue
				}
				worlds[i]["workspaces"] = []map[string]string{
					{"name": "default", "path": wsStr},
				}
				delete(worlds[i], "workspace")
				changed = true
			}

			if !changed {
				return nil
			}

			out, err := json.MarshalIndent(worlds, "", "  ")
			if err != nil {
				return err
			}
			return os.WriteFile(path, out, 0644)
		},
	}
}
