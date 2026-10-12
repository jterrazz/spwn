package user

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"spwn.sh/packages/migration"

	"gopkg.in/yaml.v3"
)

// RestructureAgentDirs renames core/ to identity/, flattens memory/ subdirs
// to root level, and merges sessions/ into journal/.
var RestructureAgentDirs = migration.Migration{
	Number:      11,
	Description: "restructure agent dirs: core->identity, flatten memory, merge sessions into journal",
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
			agentDir := filepath.Join(agentsDir, entry.Name())
			if err := restructureAgent(agentDir); err != nil {
				return err
			}
		}
		return nil
	},
}

func restructureAgent(agentDir string) error {
	coreDir := filepath.Join(agentDir, "core")
	identityDir := filepath.Join(agentDir, "identity")
	if dirExists(coreDir) && !dirExists(identityDir) {
		if err := os.Rename(coreDir, identityDir); err != nil {
			return err
		}
	}

	if err := moveSubdir(agentDir, filepath.Join("memory", "knowledge"), "knowledge"); err != nil {
		return err
	}

	if err := moveSubdir(agentDir, filepath.Join("memory", "playbooks"), "playbooks"); err != nil {
		return err
	}

	memJournalDir := filepath.Join(agentDir, "memory", "journal")
	journalDir := filepath.Join(agentDir, "journal")
	if dirExists(memJournalDir) {
		if err := mergeDir(memJournalDir, journalDir); err != nil {
			return err
		}
		os.RemoveAll(memJournalDir)
	}

	sessionsDir := filepath.Join(agentDir, "sessions")
	if dirExists(sessionsDir) {
		if err := mergeDir(sessionsDir, journalDir); err != nil {
			return err
		}
		os.RemoveAll(sessionsDir)
	}

	memoryDir := filepath.Join(agentDir, "memory")
	if dirExists(memoryDir) {
		removeIfEmpty(memoryDir)
	}

	if err := slimProfileYAML(agentDir); err != nil {
		return err
	}

	return nil
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func moveSubdir(agentDir, src, dst string) error {
	srcPath := filepath.Join(agentDir, src)
	dstPath := filepath.Join(agentDir, dst)

	if !dirExists(srcPath) {
		return nil
	}

	if !dirExists(dstPath) {
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}
		return os.Rename(srcPath, dstPath)
	}

	if err := mergeDir(srcPath, dstPath); err != nil {
		return err
	}
	return os.RemoveAll(srcPath)
}

func mergeDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
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

		if _, err := os.Stat(target); err == nil {
			return nil
		}

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
}

func slimProfileYAML(agentDir string) error {
	path := filepath.Join(agentDir, "profile.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil
	}

	for _, key := range []string{"identity", "requires", "delegation", "memory"} {
		delete(raw, key)
	}

	rewritten, err := yaml.Marshal(raw)
	if err != nil {
		return nil
	}

	if strings.TrimSpace(string(rewritten)) != strings.TrimSpace(string(data)) {
		return os.WriteFile(path, rewritten, 0644)
	}
	return nil
}

func removeIfEmpty(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	if len(entries) == 0 {
		_ = os.Remove(dir)
	}
}
