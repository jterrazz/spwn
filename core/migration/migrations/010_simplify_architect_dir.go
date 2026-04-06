package migrations

import (
	"context"
	"os"
	"path/filepath"

	"spwn.sh/core/migration"
)

// SimplifyArchitectDir consolidates the architect directory to a single
// stack.md file. If directives.md exists its content is appended to stack.md
// (with a separator). todo.md is removed unconditionally.
var SimplifyArchitectDir = migration.Migration{
	Number:      10,
	Description: "simplify architect/ to stack.md only",
	Apply: func(_ context.Context, baseDir string) error {
		archDir := filepath.Join(baseDir, "architect")

		// If the architect directory doesn't exist, nothing to do.
		if _, err := os.Stat(archDir); os.IsNotExist(err) {
			return nil
		}

		stackPath := filepath.Join(archDir, "stack.md")
		directivesPath := filepath.Join(archDir, "directives.md")
		todoPath := filepath.Join(archDir, "todo.md")

		// Merge directives.md into stack.md if it exists.
		if data, err := os.ReadFile(directivesPath); err == nil && len(data) > 0 {
			// Ensure stack.md exists (may be empty or absent).
			existing, _ := os.ReadFile(stackPath)

			merged := existing
			if len(merged) > 0 {
				merged = append(merged, []byte("\n\n---\n\n# Archived Directives\n\n")...)
			}
			merged = append(merged, data...)

			if err := os.WriteFile(stackPath, merged, 0644); err != nil {
				return err
			}
		}

		// Remove directives.md.
		if err := os.Remove(directivesPath); err != nil && !os.IsNotExist(err) {
			return err
		}

		// Remove todo.md.
		if err := os.Remove(todoPath); err != nil && !os.IsNotExist(err) {
			return err
		}

		return nil
	},
}
