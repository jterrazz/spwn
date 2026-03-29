package sync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitCmdFormatsCorrectly(t *testing.T) {
	// Create a temporary directory to act as a git repo
	tmp := t.TempDir()

	// Initialize a git repo in the temp dir
	if err := gitCmd(tmp, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user for the test repo
	if err := gitCmd(tmp, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}
	if err := gitCmd(tmp, "config", "user.name", "Test"); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	// Verify .git directory was created
	if _, err := os.Stat(filepath.Join(tmp, ".git")); os.IsNotExist(err) {
		t.Fatal("expected .git directory to be created")
	}

	// git status should succeed on initialized repo
	if err := gitCmd(tmp, "status"); err != nil {
		t.Fatalf("git status failed on initialized repo: %v", err)
	}
}

func TestGitCmdFailsOnInvalidArgs(t *testing.T) {
	tmp := t.TempDir()

	// Running git status on a non-repo should fail
	err := gitCmd(tmp, "status")
	if err == nil {
		t.Fatal("expected error running git status on non-repo directory")
	}
}

func TestSyncToGitWithTempRepo(t *testing.T) {
	// Create a temp dir to simulate ~/.spwn/
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	// Create a file to commit
	if err := os.WriteFile(filepath.Join(tmp, "test.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// SyncToGit with no remote should init, add, and commit
	if err := SyncToGit("", ""); err != nil {
		t.Fatalf("SyncToGit failed: %v", err)
	}

	// Verify a commit was made
	if err := gitCmd(tmp, "log", "--oneline", "-1"); err != nil {
		t.Fatalf("expected at least one commit: %v", err)
	}
}

func TestSyncToGitNoChanges(t *testing.T) {
	// Create a temp dir to simulate ~/.spwn/
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	// Initialize and make an initial commit
	if err := gitCmd(tmp, "init"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if err := gitCmd(tmp, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config: %v", err)
	}
	if err := gitCmd(tmp, "config", "user.name", "Test"); err != nil {
		t.Fatalf("git config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "file.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := gitCmd(tmp, "add", "-A"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := gitCmd(tmp, "commit", "-m", "initial"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	// SyncToGit with no new changes should succeed (no-op)
	if err := SyncToGit("", ""); err != nil {
		t.Fatalf("SyncToGit with no changes should not fail: %v", err)
	}
}
