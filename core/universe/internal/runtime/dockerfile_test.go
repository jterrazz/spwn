package runtime_test

import (
	"strings"
	"testing"

	"spwn.sh/core/universe/internal/runtime"

	// Import all adapters for registration
	_ "spwn.sh/core/universe/internal/runtime/aider"
	_ "spwn.sh/core/universe/internal/runtime/claude"
	_ "spwn.sh/core/universe/internal/runtime/codex"
	_ "spwn.sh/core/universe/internal/runtime/gemini"
	_ "spwn.sh/core/universe/internal/runtime/opencode"
	_ "spwn.sh/core/universe/internal/runtime/pi"
)

func TestGenerateDockerfileAllRuntimes(t *testing.T) {
	names := []string{"claude-code", "pi", "codex", "opencode", "gemini", "aider"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			r, err := runtime.Get(name)
			if err != nil {
				t.Fatal(err)
			}

			df := runtime.GenerateDockerfile(r)

			// Must start with FROM
			if !strings.HasPrefix(df, "FROM ") {
				t.Errorf("Dockerfile must start with FROM, got: %s", df[:50])
			}

			// Must contain base image
			if !strings.Contains(df, r.BaseImage()) {
				t.Errorf("missing base image %q", r.BaseImage())
			}

			// Must contain install commands
			for _, cmd := range r.InstallCommands() {
				if !strings.Contains(df, cmd) {
					t.Errorf("missing install command: %s", cmd)
				}
			}

			// Must create user
			if !strings.Contains(df, "useradd") {
				t.Error("missing user creation")
			}

			// Must have mount points
			for _, dir := range []string{"/workspace", "/mind", "/universe", "/world"} {
				if !strings.Contains(df, dir) {
					t.Errorf("missing mount point %s", dir)
				}
			}

			// Must end with entrypoint
			if !strings.Contains(df, "ENTRYPOINT") {
				t.Error("missing ENTRYPOINT")
			}
		})
	}
}

func TestDockerfileBaseImages(t *testing.T) {
	// Verify each runtime uses a valid base image
	expectedImages := map[string]string{
		"claude-code": "node:20",
		"pi":          "node:20",
		"codex":       "node:20",
		"opencode":    "debian:bookworm-slim",
		"gemini":      "node:20",
		"aider":       "python:3.12-slim",
	}

	for name, expectedImage := range expectedImages {
		t.Run(name, func(t *testing.T) {
			r, _ := runtime.Get(name)
			if r.BaseImage() != expectedImage {
				t.Errorf("expected %q, got %q", expectedImage, r.BaseImage())
			}
		})
	}
}
