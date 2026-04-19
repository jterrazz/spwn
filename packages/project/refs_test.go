package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidate_refKinds exercises every ref kind end-to-end through
// manifest.Validate against an on-disk project: spwn:* builtin,
// tool:<name> local, github:<owner>/<repo> remote registry, and
// bare/legacy forms that must now surface as invalid.
func TestValidate_refKinds(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "spwn.yaml"), `version: 1
name: refs-test
worlds:
  default:
    agents: [neo]
    workspaces: [.]
`)

	agentDir := filepath.Join(root, "spwn", "agents", "neo")
	mustMkdir(t, agentDir)
	writeFile(t, filepath.Join(agentDir, "SOUL.md"), "test")
	writeFile(t, filepath.Join(agentDir, "AGENTS.md"), "test")
	writeFile(t, filepath.Join(agentDir, "agent.yaml"), `runtime:
  backend: "spwn:claude-code"
dependencies:
  - "spwn:python"
  - "tool:local-tool"
  - "tool:local-missing"
  - "github:jterrazz/python"
  - "bare-legacy"
`)

	// Local tool directory that exists on disk.
	mustMkdir(t, filepath.Join(root, "spwn", "tools", "local-tool"))

	p, err := Load(filepath.Join(root, "spwn.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	issues := Validate(p, ValidateOpts{
		BuiltinTools:      []string{"spwn:python", "spwn:claude-code"},
		SupportedRuntimes: []string{"spwn:claude-code"},
	})

	var (
		spwnPythonIssues    int
		localToolIssues     int
		missingLocalMsg     string
		jterrazzMsg         string
		jterrazzHint        string
		bareInvalidMsg      string
		bareInvalidHint     string
	)
	for _, iss := range issues {
		msg := iss.Message
		switch {
		case strings.Contains(msg, `"spwn:python"`):
			spwnPythonIssues++
		case strings.Contains(msg, `"tool:local-tool"`):
			localToolIssues++
		case strings.Contains(msg, `"tool:local-missing"`):
			missingLocalMsg = msg
		case strings.Contains(msg, `"github:jterrazz/python"`):
			jterrazzMsg = msg
			jterrazzHint = iss.Hint
		case strings.Contains(msg, `"bare-legacy"`):
			bareInvalidMsg = msg
			bareInvalidHint = iss.Hint
		}
	}

	if spwnPythonIssues != 0 {
		t.Errorf("spwn:python should produce no issue, got %d", spwnPythonIssues)
	}
	if localToolIssues != 0 {
		t.Errorf("tool:local-tool (present on disk) should produce no issue, got %d", localToolIssues)
	}
	if !strings.Contains(missingLocalMsg, "does not exist") {
		t.Errorf("tool:local-missing: want generic 'does not exist' error, got %q", missingLocalMsg)
	}
	if !strings.Contains(jterrazzMsg, "remote registries are not yet supported") {
		t.Errorf("github:jterrazz/python: want registry-unsupported message, got %q", jterrazzMsg)
	}
	if !strings.Contains(jterrazzHint, "spwn:<name>") {
		t.Errorf("github:jterrazz/python: hint should mention spwn:<name>, got %q", jterrazzHint)
	}
	if !strings.Contains(bareInvalidMsg, "invalid") {
		t.Errorf("bare-legacy: want invalid-ref message, got %q", bareInvalidMsg)
	}
	if !strings.Contains(bareInvalidHint, "skill:") || !strings.Contains(bareInvalidHint, "tool:") || !strings.Contains(bareInvalidHint, "hook:") {
		t.Errorf("bare-legacy hint should mention all three local schemes, got %q", bareInvalidHint)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}
