package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidate_refKinds exercises the three ref kinds end-to-end
// through manifest.Validate against an on-disk project: local,
// @spwn/* builtin, and @<owner>/<name> remote registry.
func TestValidate_refKinds(t *testing.T) {
	root := t.TempDir()

	// Minimal project layout: spwn.yaml + one agent that references
	// every kind of tool we want to exercise.
	writeFile(t, filepath.Join(root, "spwn.yaml"), `version: 2
name: refs-test
worlds:
  default:
    agents: [neo]
    workspaces: [.]
`)

	agentDir := filepath.Join(root, "spwn", "agents", "neo")
	mustMkdir(t, filepath.Join(agentDir, "identity"))
	writeFile(t, filepath.Join(agentDir, "identity", "profile.md"), "test")
	writeFile(t, filepath.Join(agentDir, "AGENTS.md"), "test")
	writeFile(t, filepath.Join(agentDir, "agent.yaml"), `runtime:
  backend: "@spwn/claude-code"
dependencies:
  - "@spwn/python"
  - "local-tool"
  - "local-missing"
  - "@jterrazz/python"
  - "@community/sci"
`)

	// Local dependency that exists on disk.
	mustMkdir(t, filepath.Join(root, "spwn", "tools", "local-tool"))

	p, err := Load(filepath.Join(root, "spwn.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	issues := Validate(p, ValidateOpts{
		// Fake catalog so the test doesn't depend on the real one.
		BuiltinTools:      []string{"@spwn/python", "@spwn/claude-code"},
		SupportedRuntimes: []string{"@spwn/claude-code"},
	})

	// Collect tool-related messages.
	var (
		spwnPythonIssues    int
		localToolIssues     int
		missingLocalMsg     string
		jterrazzMsg         string
		jterrazzHint        string
		communityMsg        string
		registryUnsupported int
		notFoundCount       int
	)
	for _, iss := range issues {
		msg := iss.Message
		switch {
		case strings.Contains(msg, `"@spwn/python"`):
			spwnPythonIssues++
		case strings.Contains(msg, `"local-tool"`):
			localToolIssues++
		case strings.Contains(msg, `"local-missing"`):
			missingLocalMsg = msg
			notFoundCount++
		case strings.Contains(msg, `"@jterrazz/python"`):
			jterrazzMsg = msg
			jterrazzHint = iss.Hint
			registryUnsupported++
		case strings.Contains(msg, `"@community/sci"`):
			communityMsg = msg
			registryUnsupported++
		}
	}

	if spwnPythonIssues != 0 {
		t.Errorf("@spwn/python should produce no issue, got %d", spwnPythonIssues)
	}
	if localToolIssues != 0 {
		t.Errorf("local-tool (present on disk) should produce no issue, got %d", localToolIssues)
	}
	if !strings.Contains(missingLocalMsg, "does not exist") {
		t.Errorf("local-missing: want generic 'does not exist' error, got %q", missingLocalMsg)
	}
	if !strings.Contains(jterrazzMsg, "remote registries are not yet supported") {
		t.Errorf("@jterrazz/python: want registry-unsupported message, got %q", jterrazzMsg)
	}
	if !strings.Contains(jterrazzHint, "spwn:<name>") || !strings.Contains(jterrazzHint, "./spwn/tools/") {
		t.Errorf("@jterrazz/python: hint should mention both workarounds, got %q", jterrazzHint)
	}
	if !strings.Contains(communityMsg, "remote registries are not yet supported") {
		t.Errorf("@community/sci: want registry-unsupported message, got %q", communityMsg)
	}
	if registryUnsupported != 2 {
		t.Errorf("two registry refs should produce two distinct issues, got %d", registryUnsupported)
	}
	if notFoundCount != 1 {
		t.Errorf("one not-found ref expected, got %d", notFoundCount)
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
