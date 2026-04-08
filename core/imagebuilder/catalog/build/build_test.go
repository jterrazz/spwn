package build

import "testing"

func TestBuild_Name(t *testing.T) {
	if Tool.Name() != "@build" {
		t.Errorf("expected @build, got %s", Tool.Name())
	}
}

func TestBuild_HasVerify(t *testing.T) {
	if len(Tool.Verify()) != 3 {
		t.Errorf("expected 3 verify commands, got %d", len(Tool.Verify()))
	}
}
