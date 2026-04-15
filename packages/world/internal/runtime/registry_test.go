package runtime_test

import (
	"testing"

	"spwn.sh/packages/world/internal/runtime"

	_ "spwn.sh/packages/world/internal/runtime/claude"
)

func TestClaudeCodeRegistered(t *testing.T) {
	r, err := runtime.Get("claude-code")
	if err != nil {
		t.Fatalf("claude-code not registered: %v", err)
	}
	if r.Name() != "claude-code" {
		t.Errorf("expected claude-code, got %q", r.Name())
	}
}

func TestGetUnknownRuntime(t *testing.T) {
	_, err := runtime.Get("nonexistent")
	if err == nil {
		t.Error("expected error for unknown runtime, got nil")
	}
}

func TestClaudeCodeBuildCommand(t *testing.T) {
	r, err := runtime.Get("claude-code")
	if err != nil {
		t.Fatal(err)
	}

	cmd := r.BuildCommand(runtime.SpawnConfig{Prompt: "hello world"})
	if len(cmd) == 0 {
		t.Fatal("empty command")
	}
	if cmd[0] != "claude" {
		t.Errorf("expected binary %q, got %q", "claude", cmd[0])
	}

	found := false
	for _, arg := range cmd {
		if arg == "hello world" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("prompt not found in command args: %v", cmd)
	}
}

func TestClaudeCodeWithModel(t *testing.T) {
	r, err := runtime.Get("claude-code")
	if err != nil {
		t.Fatal(err)
	}

	cmd := r.BuildCommand(runtime.SpawnConfig{
		Prompt: "test",
		Model:  "test-model",
	})
	if len(cmd) == 0 {
		t.Fatal("empty command with model")
	}
}

func TestClaudeCodeMetadata(t *testing.T) {
	r, err := runtime.Get("claude-code")
	if err != nil {
		t.Fatal(err)
	}

	if r.BaseImage() == "" {
		t.Error("empty base image")
	}
	if len(r.InstallCommands()) == 0 {
		t.Error("no install commands")
	}
	if r.PrelaunchShell() == "" {
		t.Error("empty prelaunch shell (needed to source /credentials/.env)")
	}
	if len(r.DefaultConfigFiles("/agents/neo")) == 0 {
		t.Error("no default config files (needed to dismiss first-run UI)")
	}

	hasGit := false
	for _, p := range r.SystemPackages() {
		if p == "git" {
			hasGit = true
			break
		}
	}
	if !hasGit {
		t.Error("system packages missing git")
	}
}
