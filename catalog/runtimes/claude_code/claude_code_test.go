package claude_code

import (
	"io/fs"
	"testing"
)

func TestClaudeCode_Name(t *testing.T) {
	if Tool.Name() != "@spwn/claude-code" {
		t.Errorf("expected @spwn/claude-code, got %s", Tool.Name())
	}
}

func TestClaudeCode_DependsOnNode(t *testing.T) {
	deps := Tool.Dependencies()
	if len(deps) != 1 || deps[0] != "@spwn/node" {
		t.Errorf("expected [@spwn/node] dependency, got %v", deps)
	}
}

func TestClaudeCode_HasSkills(t *testing.T) {
	s := Tool.Skills()
	if s == nil {
		t.Fatal("expected skills FS")
	}
	_, err := fs.ReadFile(s, "SKILL.md")
	if err != nil {
		t.Errorf("expected SKILL.md in skills: %v", err)
	}
}

func TestClaudeCode_HasInstallCommands(t *testing.T) {
	spec := Tool.Install()
	if len(spec.Commands) == 0 {
		t.Error("expected install commands")
	}
}
