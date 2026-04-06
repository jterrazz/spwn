package migrations

import (
	"context"
	"os"
	"path/filepath"

	"spwn.sh/core/migration"
)

// RenameHierarchiesToOrganizations renames ~/.spwn/hierarchies/ to ~/.spwn/organizations/.
var RenameHierarchiesToOrganizations = migration.Migration{
	Number:      12,
	Description: "rename hierarchies/ directory to organizations/",
	Apply: func(_ context.Context, baseDir string) error {
		src := filepath.Join(baseDir, "hierarchies")
		dst := filepath.Join(baseDir, "organizations")

		// Already renamed or never existed
		if _, err := os.Stat(src); os.IsNotExist(err) {
			return nil
		}
		// Destination already exists — skip
		if _, err := os.Stat(dst); err == nil {
			return nil
		}

		return os.Rename(src, dst)
	},
}
