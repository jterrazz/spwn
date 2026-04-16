package upgrade

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackupBaseDirCreatesBackup(t *testing.T) {
	dir := t.TempDir()

	// Create some files
	os.WriteFile(filepath.Join(dir, "state.json"), []byte(`{"key":"val"}`), 0644)
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("name: test"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.md"), []byte("# Notes"), 0644)
	os.WriteFile(filepath.Join(dir, "binary.bin"), []byte("should be skipped"), 0644)

	// Create a subdirectory with a json file
	sub := filepath.Join(dir, "sub")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "nested.json"), []byte(`{}`), 0644)

	if err := BackupBaseDir(dir); err != nil {
		t.Fatalf("BackupBaseDir: %v", err)
	}

	// Check .backups dir exists
	backupRoot := filepath.Join(dir, backupSubDir)
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 backup dir, got %d", len(entries))
	}

	backupDir := filepath.Join(backupRoot, entries[0].Name())

	// Check json was copied
	if _, err := os.Stat(filepath.Join(backupDir, "state.json")); err != nil {
		t.Error("state.json not backed up")
	}
	// Check yaml was copied
	if _, err := os.Stat(filepath.Join(backupDir, "config.yaml")); err != nil {
		t.Error("config.yaml not backed up")
	}
	// Check md was copied
	if _, err := os.Stat(filepath.Join(backupDir, "notes.md")); err != nil {
		t.Error("notes.md not backed up")
	}
	// Check binary was NOT copied
	if _, err := os.Stat(filepath.Join(backupDir, "binary.bin")); !os.IsNotExist(err) {
		t.Error("binary.bin should not be backed up")
	}
	// Check nested json was copied
	if _, err := os.Stat(filepath.Join(backupDir, "sub", "nested.json")); err != nil {
		t.Error("sub/nested.json not backed up")
	}

	// Verify content is correct
	data, _ := os.ReadFile(filepath.Join(backupDir, "state.json"))
	if string(data) != `{"key":"val"}` {
		t.Errorf("backup content = %q, want %q", string(data), `{"key":"val"}`)
	}
}

func TestBackupBaseDirSkipsBackupsDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{}`), 0644)

	// Create a pre-existing .backups with something in it
	oldBackup := filepath.Join(dir, backupSubDir, "pre-migration-20200101-000000")
	os.MkdirAll(oldBackup, 0755)
	os.WriteFile(filepath.Join(oldBackup, "old.json"), []byte(`{}`), 0644)

	if err := BackupBaseDir(dir); err != nil {
		t.Fatalf("BackupBaseDir: %v", err)
	}

	// The new backup should NOT contain a .backups subdirectory
	entries, _ := os.ReadDir(filepath.Join(dir, backupSubDir))
	for _, e := range entries {
		backupPath := filepath.Join(dir, backupSubDir, e.Name())
		if _, err := os.Stat(filepath.Join(backupPath, backupSubDir)); !os.IsNotExist(err) {
			t.Error("backup should not contain .backups subdirectory")
		}
	}
}

func TestBackupPrunesToMax(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{}`), 0644)

	backupRoot := filepath.Join(dir, backupSubDir)

	// Create 4 existing backups with different timestamps
	for i := 0; i < 4; i++ {
		ts := time.Date(2025, 1, 1, 0, 0, i, 0, time.UTC).Format("20060102-150405")
		name := "pre-migration-" + ts
		os.MkdirAll(filepath.Join(backupRoot, name), 0755)
	}

	// This creates a 5th backup, prune should bring it down to 3
	if err := BackupBaseDir(dir); err != nil {
		t.Fatalf("BackupBaseDir: %v", err)
	}

	entries, _ := os.ReadDir(backupRoot)
	var count int
	for _, e := range entries {
		if e.IsDir() {
			count++
		}
	}
	if count > maxBackups {
		t.Errorf("expected at most %d backups, got %d", maxBackups, count)
	}
}
