package git

import "testing"

func TestGit_Name(t *testing.T) {
	if Tool.Name() != "@spwn/git" {
		t.Errorf("expected @spwn/git, got %s", Tool.Name())
	}
}

func TestGit_NoDependencies(t *testing.T) {
	if len(Tool.Dependencies()) != 0 {
		t.Error("@spwn/git should have no dependencies")
	}
}

func TestGit_HasVerify(t *testing.T) {
	if len(Tool.Verify()) == 0 {
		t.Error("@spwn/git should have verify commands")
	}
}
