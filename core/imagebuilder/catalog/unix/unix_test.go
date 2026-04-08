package unix

import (
	"testing"
)

func TestUnix_Name(t *testing.T) {
	if Tool.Name() != "@unix" {
		t.Errorf("expected @unix, got %s", Tool.Name())
	}
}

func TestUnix_NoDependencies(t *testing.T) {
	if len(Tool.Dependencies()) != 0 {
		t.Error("@unix should have no dependencies")
	}
}

func TestUnix_HasPackages(t *testing.T) {
	spec := Tool.Install()
	if len(spec.Packages) == 0 {
		t.Error("@unix should have packages")
	}
}

func TestUnix_HasVerify(t *testing.T) {
	if len(Tool.Verify()) == 0 {
		t.Error("@unix should have verify commands")
	}
}

func TestUnix_NoSkills(t *testing.T) {
	if Tool.Skills() != nil {
		t.Error("@unix should not have skills")
	}
}
