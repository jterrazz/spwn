package migration

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	backupSubDir = ".backups"
	maxBackups   = 3
)

// BackupBaseDir copies state files from baseDir into a timestamped backup dir.
// Skips the .backups directory itself. Keeps at most maxBackups recent backups.
func BackupBaseDir(baseDir string) error {
	backupRoot := filepath.Join(baseDir, backupSubDir)
	stamp := time.Now().UTC().Format("20060102-150405")
	dest := filepath.Join(backupRoot, "pre-migration-"+stamp)

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	err := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(baseDir, path)
		if rel == "." {
			return nil
		}
		if strings.HasPrefix(rel, backupSubDir) {
			return filepath.SkipDir
		}

		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		ext := filepath.Ext(path)
		if ext != ".json" && ext != ".yaml" && ext != ".yml" && ext != ".md" {
			return nil
		}
		return copyFile(path, target)
	})
	if err != nil {
		return fmt.Errorf("walk base dir: %w", err)
	}

	return pruneBackups(backupRoot)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func pruneBackups(backupRoot string) error {
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "pre-migration-") {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	for len(dirs) > maxBackups {
		oldest := dirs[0]
		dirs = dirs[1:]
		os.RemoveAll(filepath.Join(backupRoot, oldest))
	}
	return nil
}
