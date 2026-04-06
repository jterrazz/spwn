package migrations

import (
	"context"
	"os"
	"path/filepath"
)

// EnsureDefaultHierarchy creates the default governor/citizen hierarchy file
// if it does not already exist.
var EnsureDefaultHierarchy = migration003()

const defaultHierarchyYAML = `name: Default
description: Built-in governor/citizen hierarchy
roles:
  - name: governor
    level: 0
    can_command:
      - citizen
    max_per_world: 1
    permissions:
      - delegate
      - review
      - orchestrate
  - name: citizen
    level: 1
    reports_to: governor
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
