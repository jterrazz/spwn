package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRuleAgentDescription_flagsMissing: an agent.yaml that doesn't
// declare description: → LevelError with a fix-it hint.
func TestRuleAgentDescription_flagsMissing(t *testing.T) {
	root := t.TempDir()
	agentDir := writeAgentYAML(t, root, "neo", `name: neo

runtime:
  backend: "spwn:claude-code"
`)

	issues := ruleAgentDescription(Input{
		Root:      root,
		AgentRefs: []AgentRef{{Name: "neo", Path: agentDir, Exists: true}},
	})
	if len(issues) != 1 {
		t.Fatalf("want 1 issue, got %d: %+v", len(issues), issues)
	}
	if issues[0].Level != LevelError {
		t.Errorf("level: want Error, got %v", issues[0].Level)
	}
	if !strings.Contains(issues[0].Message, "description") {
		t.Errorf("message should mention description, got %q", issues[0].Message)
	}
	if !strings.Contains(issues[0].Hint, "description:") {
		t.Errorf("hint should show the field name, got %q", issues[0].Hint)
	}
	if !strings.HasSuffix(issues[0].Path, "#description") {
		t.Errorf("path should anchor at #description, got %q", issues[0].Path)
	}
}

// TestRuleAgentDescription_flagsEmpty: description: present but
// blank or whitespace-only is treated the same as missing.
func TestRuleAgentDescription_flagsEmpty(t *testing.T) {
	root := t.TempDir()
	agentDir := writeAgentYAML(t, root, "neo", `name: neo
description: "   "

runtime:
  backend: "spwn:claude-code"
`)

	issues := ruleAgentDescription(Input{
		Root:      root,
		AgentRefs: []AgentRef{{Name: "neo", Path: agentDir, Exists: true}},
	})
	if len(issues) != 1 {
		t.Fatalf("whitespace-only description should fail, got %d issues", len(issues))
	}
}

// TestRuleAgentDescription_passes: description present → no issue.
func TestRuleAgentDescription_passes(t *testing.T) {
	root := t.TempDir()
	agentDir := writeAgentYAML(t, root, "neo", `name: neo
description: The explorer — learns by doing.

runtime:
  backend: "spwn:claude-code"
`)

	issues := ruleAgentDescription(Input{
		Root:      root,
		AgentRefs: []AgentRef{{Name: "neo", Path: agentDir, Exists: true}},
	})
	if len(issues) != 0 {
		t.Errorf("want 0 issues, got %d: %+v", len(issues), issues)
	}
}

// TestRuleAgentDescription_skipsMissingDir: an agent referenced by
// the manifest but with no directory on disk is flagged elsewhere
// (ruleAgentDirsExist); this rule silently skips to avoid
// double-reporting.
func TestRuleAgentDescription_skipsMissingDir(t *testing.T) {
	root := t.TempDir()
	issues := ruleAgentDescription(Input{
		Root:      root,
		AgentRefs: []AgentRef{{Name: "ghost", Path: filepath.Join(root, "spwn/agents/ghost"), Exists: false}},
	})
	if len(issues) != 0 {
		t.Errorf("missing dirs should be skipped, got %d issues", len(issues))
	}
}

// TestRuleAgentDescription_skipsUnparseable: a malformed YAML file
// is already flagged by ruleAgentYAMLParses; description check
// bails out silently so the user sees one clear error, not two.
func TestRuleAgentDescription_skipsUnparseable(t *testing.T) {
	root := t.TempDir()
	agentDir := writeAgentYAML(t, root, "broken", "name: [unterminated")

	issues := ruleAgentDescription(Input{
		Root:      root,
		AgentRefs: []AgentRef{{Name: "broken", Path: agentDir, Exists: true}},
	})
	if len(issues) != 0 {
		t.Errorf("unparseable YAML should be skipped here, got %d: %+v", len(issues), issues)
	}
}

// TestRuleAgentDescription_multipleAgents: one issue per failing
// agent; agents with descriptions don't contribute issues. Lets us
// fail granularly in a colony where only some members forgot.
func TestRuleAgentDescription_multipleAgents(t *testing.T) {
	root := t.TempDir()
	neoDir := writeAgentYAML(t, root, "neo", "name: neo\ndescription: set\nruntime:\n  backend: \"spwn:claude-code\"\n")
	trinityDir := writeAgentYAML(t, root, "trinity", "name: trinity\nruntime:\n  backend: \"spwn:codex\"\n")

	issues := ruleAgentDescription(Input{
		Root: root,
		AgentRefs: []AgentRef{
			{Name: "neo", Path: neoDir, Exists: true},
			{Name: "trinity", Path: trinityDir, Exists: true},
		},
	})
	if len(issues) != 1 {
		t.Fatalf("one failing agent should produce one issue, got %d: %+v", len(issues), issues)
	}
	if !strings.Contains(issues[0].Message, "trinity") {
		t.Errorf("issue should name the failing agent, got %q", issues[0].Message)
	}
}

func writeAgentYAML(t *testing.T, root, name, body string) string {
	t.Helper()
	dir := filepath.Join(root, "spwn", "agents", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "agent.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
