package sync

import (
	"fmt"
	"os/exec"

	"spwn.sh/packages/foundation"
)

// SyncToGit commits and pushes ~/.spwn/ changes to the configured git repo.
func SyncToGit(repo, branch string) error {
	baseDir := foundation.BaseDir()

	// Check if git is initialized
	if err := gitCmd(baseDir, "status"); err != nil {
		// Initialize git repo
		if err := gitCmd(baseDir, "init"); err != nil {
			return fmt.Errorf("git init failed: %w", err)
		}
		// Configure git user for commits (safe defaults if not set globally)
		gitCmd(baseDir, "config", "user.email", "spwn@spwn.sh")
		gitCmd(baseDir, "config", "user.name", "spwn")
		if repo != "" {
			if err := gitCmd(baseDir, "remote", "add", "origin", repo); err != nil {
				return fmt.Errorf("git remote add failed: %w", err)
			}
		}
	}

	// Add all changes
	if err := gitCmd(baseDir, "add", "-A"); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Commit
	if err := gitCmd(baseDir, "commit", "-m", "spwn: auto-sync config"); err != nil {
		// No changes to commit is OK
		return nil
	}

	// Push if remote configured
	if repo != "" {
		b := branch
		if b == "" {
			b = "main"
		}
		if err := gitCmd(baseDir, "push", "origin", b); err != nil {
			return fmt.Errorf("git push failed: %w", err)
		}
	}

	return nil
}

// PullFromGit pulls latest changes from the configured git repo.
func PullFromGit(repo, branch string) error {
	baseDir := foundation.BaseDir()
	b := branch
	if b == "" {
		b = "main"
	}
	if err := gitCmd(baseDir, "pull", "origin", b); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}
	return nil
}

func gitCmd(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}
