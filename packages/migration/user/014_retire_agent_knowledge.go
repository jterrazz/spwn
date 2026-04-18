package user

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"spwn.sh/packages/migration"
)

// RetireAgentKnowledge removes the now-unused knowledge/ directory
// from every agent in ~/.spwn/agents/. Knowledge moved to the world
// in 2026-04 — it's environmental (about the domain), not about the
// agent's personality. The world owns /world/knowledge/, bind-mounted
// from the project tree at spwn/worlds/<name>/knowledge/.
//
// If an agent's knowledge/ dir contains real files (not just a
// .gitkeep), they are NOT auto-moved: we don't know which world
// should inherit them. Instead the migration renames the directory
// to knowledge.retired-<timestamp>/ so the files survive and the
// user can move them into the right spwn/worlds/<name>/knowledge/
// manually. Empty directories (just .gitkeep or truly empty) are
// removed silently.
var RetireAgentKnowledge = migration.Migration{
	Number:      14,
	Description: "retire agents/<name>/knowledge/ (knowledge moved to world scope)",
	Apply: func(_ context.Context, baseDir string) error {
		agentsDir := filepath.Join(baseDir, "agents")
		if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
			return nil
		}

		entries, err := os.ReadDir(agentsDir)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			knowledgeDir := filepath.Join(agentsDir, entry.Name(), "knowledge")
			info, statErr := os.Stat(knowledgeDir)
			if statErr != nil || !info.IsDir() {
				continue
			}

			empty, err := isTriviallyEmpty(knowledgeDir)
			if err != nil {
				return fmt.Errorf("inspect %s: %w", knowledgeDir, err)
			}
			if empty {
				if err := os.RemoveAll(knowledgeDir); err != nil {
					return fmt.Errorf("remove %s: %w", knowledgeDir, err)
				}
				continue
			}

			retired := knowledgeDir + ".retired"
			if _, err := os.Stat(retired); err == nil {
				retired = fmt.Sprintf("%s.%d", retired, info.ModTime().Unix())
			}
			if err := os.Rename(knowledgeDir, retired); err != nil {
				return fmt.Errorf("rename %s → %s: %w", knowledgeDir, retired, err)
			}
		}
		return nil
	},
}

// isTriviallyEmpty reports whether a directory has nothing worth
// preserving — either truly empty, or contains only .gitkeep and/or
// empty subdirectories.
func isTriviallyEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if e.IsDir() {
			sub := filepath.Join(dir, e.Name())
			inner, err := isTriviallyEmpty(sub)
			if err != nil {
				return false, err
			}
			if !inner {
				return false, nil
			}
			continue
		}
		if e.Name() == ".gitkeep" {
			continue
		}
		return false, nil
	}
	return true, nil
}
