package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_AGENTSMDIsRuntimeNeutral(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir, Opts{Name: "neutral"}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	body := readString(t, filepath.Join(dir, "agents", "neo", "AGENTS.md"))
	for _, forbidden := range []string{"@SOUL.md", "CLAUDE.md"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("AGENTS.md contains provider-specific %q:\n%s", forbidden, body)
		}
	}
	for _, want := range []string{"SOUL.md", "runtime-native prompt", "runtime prompt"} {
		if !strings.Contains(body, want) {
			t.Fatalf("AGENTS.md missing %q:\n%s", want, body)
		}
	}
}

func TestInit_BackendPinIsWrittenWhenRequested(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir, Opts{Name: "codex-project", Backend: "spwn:codex"}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	body := readString(t, filepath.Join(dir, "agents", "neo", "agent.yaml"))
	for _, want := range []string{`runtime:`, `backend: "spwn:codex"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("agent.yaml missing %q:\n%s", want, body)
		}
	}
}

func TestInit_WritesExpectedStarterFiles(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir, Opts{Name: "starter"}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	for _, rel := range []string{
		"spwn.yaml",
		"spwn.lock",
		"agents/neo/agent.yaml",
		"agents/neo/AGENTS.md",
		"agents/neo/SOUL.md",
		"agents/neo/playbooks/.gitkeep",
		"agents/neo/journal/.gitkeep",
		"skills/focus.md",
		"tools/greet/tool.yaml",
		"hooks/session-banner.yaml",
		"commands/refactor.md",
		"knowledge/.gitkeep",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("expected scaffold file %s: %v", rel, err)
		}
	}
}

func readString(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}
