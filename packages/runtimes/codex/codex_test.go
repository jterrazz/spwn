package codex

import (
	"testing"
)

func TestCodex_Name(t *testing.T) {
	if Tool.Name() != "spwn:codex" {
		t.Errorf("expected spwn:codex, got %s", Tool.Name())
	}
}

func TestCodex_DependsOnNode(t *testing.T) {
	deps := Tool.Dependencies()
	if len(deps) != 1 || deps[0] != "spwn:node" {
		t.Errorf("expected [spwn:node] dependency, got %v", deps)
	}
}


func TestCodex_HasInstallCommand(t *testing.T) {
	spec := Tool.Install()
	if len(spec.Commands) == 0 {
		t.Error("expected install commands")
	}
}

