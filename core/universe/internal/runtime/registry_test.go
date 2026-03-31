package runtime_test

import (
	"strings"
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

var allRuntimes = []string{"claude-code", "pi", "codex", "opencode", "gemini", "aider"}

func TestAllRuntimesRegistered(t *testing.T) {
	for _, name := range allRuntimes {
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

func TestGetUnknownRuntime(t *testing.T) {
	_, err := runtime.Get("nonexistent")
	if err == nil {
		t.Error("expected error for unknown runtime, got nil")
	}
}

func TestAllRuntimesBuildCommand(t *testing.T) {
	expectedBins := map[string]string{
		"claude-code": "claude",
		"pi":          "pi",
		"codex":       "codex",
		"opencode":    "opencode",
		"gemini":      "gemini",
		"aider":       "aider",
	}

	for _, name := range allRuntimes {
		t.Run(name, func(t *testing.T) {
			r, err := runtime.Get(name)
			if err != nil {
				t.Fatal(err)
			}

			// One-shot with prompt
			cmd := r.BuildCommand(runtime.SpawnConfig{Prompt: "hello world"})
			if len(cmd) == 0 {
				t.Fatal("empty command")
			}
			if cmd[0] == "" {
				t.Fatal("empty binary name")
			}

			// Verify binary name matches expectations
			if cmd[0] != expectedBins[name] {
				t.Errorf("expected binary %q, got %q", expectedBins[name], cmd[0])
			}

			// Verify prompt appears somewhere in command args
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
		})
	}
}

func TestAllRuntimesWithModel(t *testing.T) {
	for _, name := range allRuntimes {
		t.Run(name, func(t *testing.T) {
			r, err := runtime.Get(name)
			if err != nil {
				t.Fatal(err)
			}

			// Should not panic with model set
			cmd := r.BuildCommand(runtime.SpawnConfig{
				Prompt: "test",
				Model:  "test-model",
			})
			if len(cmd) == 0 {
				t.Fatal("empty command with model")
			}
		})
	}
}

func TestAllRuntimesNPCMode(t *testing.T) {
	for _, name := range allRuntimes {
		t.Run(name, func(t *testing.T) {
			r, err := runtime.Get(name)
			if err != nil {
				t.Fatal(err)
			}

			// NPC mode: no MindPath
			cmd := r.BuildCommand(runtime.SpawnConfig{
				Prompt: "do something",
			})
			if len(cmd) == 0 {
				t.Fatal("empty command in NPC mode")
			}

			// Should not contain session-related flags when no MindPath
			for _, arg := range cmd {
				if arg == "--session-id" {
					t.Error("NPC mode should not include --session-id")
				}
			}
		})
	}
}

func TestAllRuntimesMetadata(t *testing.T) {
	for _, name := range allRuntimes {
		t.Run(name, func(t *testing.T) {
			r, err := runtime.Get(name)
			if err != nil {
				t.Fatal(err)
			}

			if r.BaseImage() == "" {
				t.Error("empty base image")
			}
			if len(r.InstallCommands()) == 0 {
				t.Error("no install commands")
			}

			// At least one auth mechanism
			if len(r.RequiredEnvVars()) == 0 && len(r.OptionalEnvVars()) == 0 {
				t.Error("no env vars defined (need at least optional auth)")
			}

			// System packages should contain git
			pkgs := r.SystemPackages()
			hasGit := false
			for _, p := range pkgs {
				if p == "git" {
					hasGit = true
					break
				}
			}
			if !hasGit {
				t.Error("system packages missing git")
			}
		})
	}
}

func TestGenerateDockerfile(t *testing.T) {
	for _, name := range allRuntimes {
		t.Run(name, func(t *testing.T) {
			r, err := runtime.Get(name)
			if err != nil {
				t.Fatal(err)
			}

			df := runtime.GenerateDockerfile(r)

			if !strings.HasPrefix(df, "FROM ") {
				t.Error("missing FROM directive")
			}
			if !strings.Contains(df, r.BaseImage()) {
				t.Errorf("missing base image %s", r.BaseImage())
			}
			if !strings.Contains(df, "useradd") {
				t.Error("missing user creation")
			}
			if !strings.Contains(df, "/workspace") {
				t.Error("missing workspace volume")
			}

			// Verify install commands are present
			for _, cmd := range r.InstallCommands() {
				if !strings.Contains(df, cmd) {
					t.Errorf("missing install command: %s", cmd)
				}
			}

			// Verify system packages appear in apt-get install line
			for _, pkg := range r.SystemPackages() {
				if !strings.Contains(df, pkg) {
					t.Errorf("missing system package: %s", pkg)
				}
			}
		})
	}
}
