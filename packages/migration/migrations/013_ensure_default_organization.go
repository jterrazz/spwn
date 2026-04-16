package migrations

import (
	"context"
	"os"
	"path/filepath"

	"spwn.sh/packages/migration"
)

// EnsureDefaultOrganization creates the default organization YAML in the
// organizations/ directory if it doesn't exist. This covers the case where
// migration 003 wrote to hierarchies/ and migration 012 renamed it, but
// also handles fresh installs.
var EnsureDefaultOrganization = migration.Migration{
	Number:      13,
	Description: "ensure default organization file exists",
	Apply: func(_ context.Context, baseDir string) error {
		dir := filepath.Join(baseDir, "organizations")
		path := filepath.Join(dir, "default.yaml")

		if _, err := os.Stat(path); err == nil {
			return nil // already exists
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		content := `name: Default
description: Built-in three-tier organization
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
		return os.WriteFile(path, []byte(content), 0644)
	},
}
