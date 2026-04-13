package architect

import (
	"io/fs"
	"testing"
)

func TestArchitect_Name(t *testing.T) {
	if Tool.Name() != "@spwn/architect" {
		t.Errorf("expected @spwn/architect, got %s", Tool.Name())
	}
}

func TestArchitect_Dependencies(t *testing.T) {
	deps := Tool.Dependencies()
	expected := map[string]bool{"@spwn/cli": true, "@spwn/claude-code": true, "@spwn/docker-cli": true}
	if len(deps) != 3 {
		t.Fatalf("expected 3 deps, got %d", len(deps))
	}
	for _, d := range deps {
		if !expected[d] {
			t.Errorf("unexpected dependency: %s", d)
		}
	}
}

func TestArchitect_HasSkills(t *testing.T) {
	s := Tool.Skills()
	if s == nil {
		t.Fatal("expected skills FS")
	}

	expectedFiles := []string{"SKILL.md", "fleet-ops.md", "task-planning.md", "monitoring.md"}
	for _, f := range expectedFiles {
		_, err := fs.ReadFile(s, f)
		if err != nil {
			t.Errorf("expected %s in skills: %v", f, err)
		}
	}
}

func TestArchitect_HasEntrypoint(t *testing.T) {
	spec := Tool.Install()
	if spec.Files == nil {
		t.Fatal("expected files in install spec")
	}
	if _, ok := spec.Files["/usr/local/bin/architect-entrypoint.sh"]; !ok {
		t.Error("expected entrypoint.sh in install files")
	}
}
