package migrations

import (
	"context"
	"os"
	"path/filepath"
)

// RemoveOrphanedUniverses removes the legacy universes/ directory if it exists.
var RemoveOrphanedUniverses = migration008()

func migration008() Migration {
	return Migration{
		Number:      8,
		Description: "remove orphaned universes/ directory from worlds rename",
		Apply: func(_ context.Context, baseDir string) error {
			dir := filepath.Join(baseDir, "universes")
			info, err := os.Stat(dir)
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if !info.IsDir() {
				return nil
			}
			return os.RemoveAll(dir)
		},
	}
}
