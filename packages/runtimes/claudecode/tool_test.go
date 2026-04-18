package claudecode

import (
	"testing"
)

func TestClaudeCode_Name(t *testing.T) {
	if Tool.Name() != "spwn:claude-code" {
		t.Errorf("expected spwn:claude-code, got %s", Tool.Name())
	}
}

func TestClaudeCode_DependsOnUnix(t *testing.T) {
	// Native install uses curl + jq (from spwn:unix) to pull the
	// bootstrap script. No Node.js runtime needed.
	deps := Tool.Dependencies()
	if len(deps) != 1 || deps[0] != "spwn:unix" {
		t.Errorf("expected [spwn:unix] dependency, got %v", deps)
	}
}


func TestClaudeCode_HasInstallCommands(t *testing.T) {
	spec := Tool.Install()
	if len(spec.Commands) == 0 {
		t.Error("expected install commands")
	}
}
