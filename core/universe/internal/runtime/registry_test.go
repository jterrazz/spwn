package runtime_test

import (
	"testing"

	"spwn.sh/core/universe/internal/runtime"

	// Register all runtime adapters
	_ "spwn.sh/core/universe/internal/runtime/aider"
	_ "spwn.sh/core/universe/internal/runtime/claude"
	_ "spwn.sh/core/universe/internal/runtime/codex"
	_ "spwn.sh/core/universe/internal/runtime/gemini"
	_ "spwn.sh/core/universe/internal/runtime/opencode"
	_ "spwn.sh/core/universe/internal/runtime/pi"
)

func TestAllRuntimesRegistered(t *testing.T) {
	expected := []string{"claude-code", "pi", "codex", "opencode", "gemini", "aider"}
	for _, name := range expected {
		r, err := runtime.Get(name)
		if err != nil {
			t.Errorf("runtime %q not registered: %v", name, err)
			continue
		}
		if r.Name() != name {
			t.Errorf("expected %q, got %q", name, r.Name())
		}
	}
}

func TestRuntimeBuildCommand(t *testing.T) {
	for _, name := range []string{"claude-code", "pi", "codex", "opencode", "gemini", "aider"} {
		r, _ := runtime.Get(name)
		cmd := r.BuildCommand(runtime.SpawnConfig{Prompt: "hello"})
		if len(cmd) == 0 {
			t.Errorf("%s returned empty command", name)
		}
	}
}

func TestRuntimeMetadata(t *testing.T) {
	for _, name := range []string{"claude-code", "pi", "codex", "opencode", "gemini", "aider"} {
		r, _ := runtime.Get(name)
		if r.BaseImage() == "" {
			t.Errorf("%s has no base image", name)
		}
		if len(r.InstallCommands()) == 0 {
			t.Errorf("%s has no install commands", name)
		}
	}
}

func TestGetUnknownRuntime(t *testing.T) {
	_, err := runtime.Get("nonexistent")
	if err == nil {
		t.Error("expected error for unknown runtime, got nil")
	}
}
