package migrations

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
)

// RenameDefaultRoles renames governor→chief and citizen→worker in state.json,
// agent profile.yaml files, and rewrites the default hierarchy file.
var RenameDefaultRoles = migration005()

const newDefaultHierarchyYAML = `name: Default
description: Built-in three-tier hierarchy
roles:
  - name: chief
    level: 0
    can_command:
      - manager
      - worker
    max_per_world: 1
    permissions:
      - delegate
      - review
      - orchestrate
  - name: manager
    level: 1
    can_command:
      - worker
    reports_to: chief
    permissions:
      - delegate
      - review
      - execute
  - name: worker
    level: 2
    reports_to: manager
    permissions:
      - execute
      - report
`

func migration005() Migration {
	return Migration{
		Number:      5,
		Description: "rename governor/citizen roles to chief/worker",
		Apply: func(_ context.Context, baseDir string) error {
			// 1. Rename roles in state.json
			statePath := filepath.Join(baseDir, "state.json")
			if data, err := os.ReadFile(statePath); err == nil {
				updated := data
				updated = bytes.ReplaceAll(updated, []byte(`"governor"`), []byte(`"chief"`))
				updated = bytes.ReplaceAll(updated, []byte(`"citizen"`), []byte(`"worker"`))
				if !bytes.Equal(updated, data) {
					if err := os.WriteFile(statePath, updated, 0644); err != nil {
						return err
					}
				}
			}

			// 2. Rename roles in agent profile.yaml files
			agentsDir := filepath.Join(baseDir, "agents")
			entries, err := os.ReadDir(agentsDir)
			if err == nil {
				for _, entry := range entries {
					if !entry.IsDir() {
						continue
					}
					profilePath := filepath.Join(agentsDir, entry.Name(), "profile.yaml")
					data, err := os.ReadFile(profilePath)
					if err != nil {
						continue
					}
					content := string(data)
					updated := content
					updated = strings.ReplaceAll(updated, "role: governor", "role: chief")
					updated = strings.ReplaceAll(updated, "role: citizen", "role: worker")
					if updated != content {
						if err := os.WriteFile(profilePath, []byte(updated), 0644); err != nil {
							return err
						}
					}
				}
			}

			// 3. Rewrite default hierarchy file with new roles
			hierPath := filepath.Join(baseDir, "hierarchies", "default.yaml")
			if _, err := os.Stat(hierPath); err == nil {
				if err := os.WriteFile(hierPath, []byte(newDefaultHierarchyYAML), 0644); err != nil {
					return err
				}
			}

			return nil
		},
	}
}
