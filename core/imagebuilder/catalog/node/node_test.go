package node

import "testing"

func TestNode_Name(t *testing.T) {
	if Tool.Name() != "@node" {
		t.Errorf("expected @node, got %s", Tool.Name())
	}
}

func TestNode_NoDependencies(t *testing.T) {
	if len(Tool.Dependencies()) != 0 {
		t.Error("@node should have no dependencies")
	}
}

func TestNode_HasCommands(t *testing.T) {
	spec := Tool.Install()
	if len(spec.Commands) == 0 {
		t.Error("@node should have install commands")
	}
}

func TestNode_VerifiesThreeBinaries(t *testing.T) {
	if len(Tool.Verify()) != 3 {
		t.Errorf("expected 3 verify commands (node, npm, npx), got %d", len(Tool.Verify()))
	}
}
