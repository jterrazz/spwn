package migrations

import (
	"context"
	"os"
	"path/filepath"
)

// EnsureDefaultHierarchy creates the default chief/manager/worker hierarchy file
// if it does not already exist.
var EnsureDefaultHierarchy = migration003()

const defaultHierarchyYAML = `name: Default
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

func migration003() Migration {
	return Migration{
		Number:      3,
		Description: "ensure default hierarchy file exists",
		Apply: func(_ context.Context, baseDir string) error {
			dir := filepath.Join(baseDir, "hierarchies")
			path := filepath.Join(dir, "default.yaml")
			if _, err := os.Stat(path); err == nil {
				return nil // already exists
			}
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			return os.WriteFile(path, []byte(defaultHierarchyYAML), 0644)
		},
	}
}
