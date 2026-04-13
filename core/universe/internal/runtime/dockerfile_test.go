package runtime_test

import (
	"strings"
	"testing"

	"spwn.sh/core/universe/internal/runtime"

	_ "spwn.sh/core/universe/internal/runtime/claude"
)

func TestGenerateDockerfileClaudeCode(t *testing.T) {
	r, err := runtime.Get("claude-code")
	if err != nil {
		t.Fatal(err)
	}

	df := runtime.GenerateDockerfile(r)

	if !strings.HasPrefix(df, "FROM ") {
		t.Errorf("Dockerfile must start with FROM, got: %s", df[:50])
	}

	if !strings.Contains(df, r.BaseImage()) {
		t.Errorf("missing base image %q", r.BaseImage())
	}

	for _, cmd := range r.InstallCommands() {
		if !strings.Contains(df, cmd) {
			t.Errorf("missing install command: %s", cmd)
		}
	}

	if !strings.Contains(df, "useradd") {
		t.Error("missing user creation")
	}

	for _, dir := range []string{"/workspace", "/mind", "/universe", "/world"} {
		if !strings.Contains(df, dir) {
			t.Errorf("missing mount point %s", dir)
		}
	}

	if !strings.Contains(df, "ENTRYPOINT") {
		t.Error("missing ENTRYPOINT")
	}
}

func TestDockerfileBaseImage(t *testing.T) {
	r, _ := runtime.Get("claude-code")
	if r.BaseImage() != "node:20" {
		t.Errorf("expected node:20, got %q", r.BaseImage())
	}
}
