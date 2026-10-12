package user

import (
	"context"
	"os"
	"path/filepath"

	"spwn.sh/packages/migration"
)

// RenameHierarchiesToOrganizations renames ~/.spwn/hierarchies/ to ~/.spwn/organizations/.
var RenameHierarchiesToOrganizations = migration.Migration{
	Number:      12,
	Description: "rename hierarchies/ directory to organizations/",
	Apply: func(_ context.Context, baseDir string) error {
		src := filepath.Join(baseDir, "hierarchies")
		dst := filepath.Join(baseDir, "organizations")

		if _, err := os.Stat(src); os.IsNotExist(err) {
			return nil
		}
		if _, err := os.Stat(dst); err == nil {
			return nil
		}

		return os.Rename(src, dst)
	},
}
