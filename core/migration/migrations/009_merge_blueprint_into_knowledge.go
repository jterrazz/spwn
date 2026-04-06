package migrations

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"spwn.sh/core/migration"
)

// MergeBlueprintIntoKnowledge moves files from the legacy blueprint/ directory
// into knowledge/ (blueprint was the old name for knowledge). Files that already
// exist in knowledge/ are skipped. The blueprint/ directory is removed afterward
// if empty.
var MergeBlueprintIntoKnowledge = migration.Migration{
	Number:      9,
	Description: "merge legacy blueprint/ directory into knowledge/",
	Apply: func(_ context.Context, baseDir string) error {
		src := filepath.Join(baseDir, "blueprint")
		dst := filepath.Join(baseDir, "knowledge")

		if _, err := os.Stat(src); os.IsNotExist(err) {
			return nil // nothing to migrate
		}

		// Ensure knowledge/ exists
		if err := os.MkdirAll(dst, 0755); err != nil {
			return err
		}

		// Walk blueprint/ and copy files that don't exist in knowledge/
		err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(src, path)
			if rel == "." {
				return nil
			}
			target := filepath.Join(dst, rel)

			if d.IsDir() {
				return os.MkdirAll(target, 0755)
			}

			// Skip if destination already exists
			if _, err := os.Stat(target); err == nil {
				return nil
			}

			// Copy file
			in, err := os.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			defer out.Close()
			_, err = io.Copy(out, in)
			return err
		})
		if err != nil {
			return err
		}

		// Remove blueprint/ entirely
		return os.RemoveAll(src)
	},
}
