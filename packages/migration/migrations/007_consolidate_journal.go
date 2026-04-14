package migrations

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
)

// ConsolidateJournal moves legacy root-level journal/ directories into
// memory/journal/ for each agent.
var ConsolidateJournal = migration007()

func migration007() Migration {
	return Migration{
		Number:      7,
		Description: "consolidate legacy root journal/ into memory/journal/ per agent",
		Apply: func(_ context.Context, baseDir string) error {
			agentsDir := filepath.Join(baseDir, "agents")
			entries, err := os.ReadDir(agentsDir)
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}

			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				agentDir := filepath.Join(agentsDir, e.Name())
				legacyJournal := filepath.Join(agentDir, "journal")
				memoryJournal := filepath.Join(agentDir, "memory", "journal")

				legacyInfo, legacyErr := os.Stat(legacyJournal)
				if legacyErr != nil || !legacyInfo.IsDir() {
					continue // no legacy journal - nothing to do
				}

				_, memoryErr := os.Stat(memoryJournal)
				memoryExists := memoryErr == nil

				if memoryExists {
					// Both exist: move files from legacy into memory/journal, skip duplicates
					if err := moveFiles(legacyJournal, memoryJournal); err != nil {
						return err
					}
					// Remove legacy dir if now empty
					removeIfEmpty(legacyJournal)
				} else {
					// Only legacy exists: rename it
					if err := os.MkdirAll(filepath.Join(agentDir, "memory"), 0755); err != nil {
						return err
					}
					if err := os.Rename(legacyJournal, memoryJournal); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

// moveFiles copies all files from src into dst, skipping files that already
// exist in dst (by filename).
func moveFiles(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, rel)

		// Skip if destination already has this file
		if _, err := os.Stat(dstPath); err == nil {
			// Remove the source copy since destination already exists
			os.Remove(path)
			return nil
		}

		// Ensure destination subdirectory exists
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		// Move (rename) the file
		if err := os.Rename(path, dstPath); err != nil {
			// Cross-device fallback: copy + remove
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if writeErr := os.WriteFile(dstPath, data, 0644); writeErr != nil {
				return writeErr
			}
			os.Remove(path)
		}
		return nil
	})
}

// removeIfEmpty removes a directory only if it contains no entries.
func removeIfEmpty(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	if len(entries) == 0 {
		os.Remove(dir)
	}
}
