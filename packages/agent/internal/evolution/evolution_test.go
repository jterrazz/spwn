package evolution

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"spwn.sh/packages/agent/internal/journal"
)

// --- Reflect tests ---

func TestReflect_WithEntries(t *testing.T) {
	mindPath := t.TempDir()

	// Create journal entries
	journal.Append(mindPath, "universe-1", 0, 5*time.Minute)
	journal.Append(mindPath, "universe-2", 1, 3*time.Minute)
	journal.Append(mindPath, "universe-3", 0, 10*time.Minute)

	result, err := Reflect(mindPath)
	if err != nil {
		t.Fatalf("Reflect() error: %v", err)
	}
	if result.Skipped {
		t.Fatal("expected Reflect to not skip")
	}
	if result.EntriesAnalyzed != 3 {
		t.Errorf("EntriesAnalyzed = %d, want 3", result.EntriesAnalyzed)
	}
	if result.CompletedTasks != 2 {
		t.Errorf("CompletedTasks = %d, want 2", result.CompletedTasks)
	}
	if result.FailedTasks != 1 {
		t.Errorf("FailedTasks = %d, want 1", result.FailedTasks)
	}

	// Verify output file exists
	if _, err := os.Stat(result.OutputPath); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

func TestReflect_EmptyJournal(t *testing.T) {
	mindPath := t.TempDir()

	result, err := Reflect(mindPath)
	if err != nil {
		t.Fatalf("Reflect() error: %v", err)
	}
	if !result.Skipped {
		t.Error("expected Reflect to skip with empty journal")
	}
	if result.Reason != "no journal entries" {
		t.Errorf("Reason = %q, want %q", result.Reason, "no journal entries")
	}
}

// --- Sleep tests ---

func TestSleep_StaleFiles(t *testing.T) {
	mindPath := t.TempDir()

	// Create a playbook with old modification time
	playbooksDir := filepath.Join(mindPath, "playbooks")
	os.MkdirAll(playbooksDir, 0755)
	staleFile := filepath.Join(playbooksDir, "old-playbook.md")
	os.WriteFile(staleFile, []byte("stale"), 0644)

	// Set modification time to 60 days ago
	oldTime := time.Now().Add(-60 * 24 * time.Hour)
	os.Chtimes(staleFile, oldTime, oldTime)

	result, err := Sleep(mindPath)
	if err != nil {
		t.Fatalf("Sleep() error: %v", err)
	}
	if result.ArchivedPlaybooks != 1 {
		t.Errorf("ArchivedPlaybooks = %d, want 1", result.ArchivedPlaybooks)
	}

	// Verify file was moved to archive
	archived := filepath.Join(mindPath, "archive", "playbooks", "old-playbook.md")
	if _, err := os.Stat(archived); err != nil {
		t.Errorf("archived file not found: %v", err)
	}
	if _, err := os.Stat(staleFile); !os.IsNotExist(err) {
		t.Error("original stale file should have been removed")
	}
}

func TestSleep_FreshFiles(t *testing.T) {
	mindPath := t.TempDir()

	// Create a fresh playbook
	playbooksDir := filepath.Join(mindPath, "playbooks")
	os.MkdirAll(playbooksDir, 0755)
	freshFile := filepath.Join(playbooksDir, "fresh-playbook.md")
	os.WriteFile(freshFile, []byte("fresh"), 0644)

	result, err := Sleep(mindPath)
	if err != nil {
		t.Fatalf("Sleep() error: %v", err)
	}
	if result.ArchivedPlaybooks != 0 {
		t.Errorf("ArchivedPlaybooks = %d, want 0", result.ArchivedPlaybooks)
	}

	// Verify file was NOT moved
	if _, err := os.Stat(freshFile); err != nil {
		t.Errorf("fresh file should still exist: %v", err)
	}
}

func TestSleep_PruneOldSessions(t *testing.T) {
	mindPath := t.TempDir()

	sessionsDir := filepath.Join(mindPath, "journal")
	os.MkdirAll(sessionsDir, 0755)

	// Create old session
	oldSession := filepath.Join(sessionsDir, "old-session.json")
	os.WriteFile(oldSession, []byte("{}"), 0644)
	oldTime := time.Now().Add(-100 * 24 * time.Hour)
	os.Chtimes(oldSession, oldTime, oldTime)

	// Create fresh session
	freshSession := filepath.Join(sessionsDir, "fresh-session.json")
	os.WriteFile(freshSession, []byte("{}"), 0644)

	result, err := Sleep(mindPath)
	if err != nil {
		t.Fatalf("Sleep() error: %v", err)
	}
	if result.PrunedSessions != 1 {
		t.Errorf("PrunedSessions = %d, want 1", result.PrunedSessions)
	}
	if _, err := os.Stat(oldSession); !os.IsNotExist(err) {
		t.Error("old session should have been pruned")
	}
	if _, err := os.Stat(freshSession); err != nil {
		t.Error("fresh session should still exist")
	}
}

// --- Fork tests ---

func TestFork_AllLayers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SPWN_HOME", home)

	// Create source agent with some layers
	sourceDir := filepath.Join(home, "agents", "source-agent")
	for _, layer := range []string{"identity", "skills", "playbooks", "journal"} {
		os.MkdirAll(filepath.Join(sourceDir, layer), 0755)
	}
	os.WriteFile(filepath.Join(sourceDir, "identity", "profile.md"), []byte("# Test"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "skills", "coding.md"), []byte("# Coding"), 0644)

	result, err := Fork("source-agent", "target-agent", nil)
	if err != nil {
		t.Fatalf("Fork() error: %v", err)
	}
	if result.Source != "source-agent" {
		t.Errorf("Source = %q, want %q", result.Source, "source-agent")
	}
	if result.Target != "target-agent" {
		t.Errorf("Target = %q, want %q", result.Target, "target-agent")
	}

	// Verify files exist in target
	targetProfile := filepath.Join(home, "agents", "target-agent", "identity", "profile.md")
	if _, err := os.Stat(targetProfile); err != nil {
		t.Errorf("target profile not found: %v", err)
	}
	targetSkill := filepath.Join(home, "agents", "target-agent", "skills", "coding.md")
	if _, err := os.Stat(targetSkill); err != nil {
		t.Errorf("target skill not found: %v", err)
	}
}

func TestFork_SpecificLayers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SPWN_HOME", home)

	// Create source agent
	sourceDir := filepath.Join(home, "agents", "source-agent")
	for _, layer := range []string{"identity", "skills", "playbooks"} {
		os.MkdirAll(filepath.Join(sourceDir, layer), 0755)
	}
	os.WriteFile(filepath.Join(sourceDir, "identity", "profile.md"), []byte("# Test"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "skills", "coding.md"), []byte("# Coding"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "playbooks", "deploy.md"), []byte("# Deploy"), 0644)

	// Fork only identity layer
	result, err := Fork("source-agent", "target-agent", []string{"identity"})
	if err != nil {
		t.Fatalf("Fork() error: %v", err)
	}

	// Verify identity was copied
	targetProfile := filepath.Join(home, "agents", "target-agent", "identity", "profile.md")
	if _, err := os.Stat(targetProfile); err != nil {
		t.Errorf("target profile not found: %v", err)
	}

	// Verify skills was NOT copied (no file inside)
	targetSkill := filepath.Join(home, "agents", "target-agent", "skills", "coding.md")
	if _, err := os.Stat(targetSkill); !os.IsNotExist(err) {
		t.Error("skills should not have been copied when only identity specified")
	}

	if len(result.LayersCopied) == 0 {
		t.Fatal("expected at least one layer copied")
	}
}

func TestFork_SourceNotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SPWN_HOME", home)

	_, err := Fork("nonexistent", "target", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}

func TestFork_TargetAlreadyExists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SPWN_HOME", home)

	// Create both source and target
	os.MkdirAll(filepath.Join(home, "agents", "source"), 0755)
	os.MkdirAll(filepath.Join(home, "agents", "target"), 0755)

	_, err := Fork("source", "target", nil)
	if err == nil {
		t.Fatal("expected error when target already exists")
	}
}
