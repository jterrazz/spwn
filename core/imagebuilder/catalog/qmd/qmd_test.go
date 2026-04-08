package qmd

import (
	"io/fs"
	"testing"
)

func TestQmd_Name(t *testing.T) {
	if Tool.Name() != "@spwn/qmd" {
		t.Errorf("expected @spwn/qmd, got %s", Tool.Name())
	}
}

func TestQmd_DependsOnNode(t *testing.T) {
	deps := Tool.Dependencies()
	if len(deps) != 1 || deps[0] != "@spwn/node" {
		t.Errorf("expected [@spwn/node] dependency, got %v", deps)
	}
}

func TestQmd_HasSkills(t *testing.T) {
	s := Tool.Skills()
	if s == nil {
		t.Fatal("expected skills FS")
	}
	_, err := fs.ReadFile(s, "SKILL.md")
	if err != nil {
		t.Errorf("expected SKILL.md: %v", err)
	}
}

func TestQmd_HasInstallCommand(t *testing.T) {
	spec := Tool.Install()
	if len(spec.Commands) == 0 {
		t.Error("expected install commands")
	}
}
