package imagetest

import "spwn.sh/packages/dependency/tool"

import (
	"io/fs"
	"strings"
	"testing"
)

// AssertValidTool checks all interface invariants on a tool.
func AssertValidTool(t *testing.T, tl tool.Tool) {
	t.Helper()

	if !strings.HasPrefix(tl.Name(), "@") {
		t.Errorf("name %q must start with @", tl.Name())
	}

	if tl.Version() == "" {
		t.Error("version must not be empty")
	}

	spec := tl.Install()
	if len(spec.AptPackages) == 0 && len(spec.Commands) == 0 && len(spec.Files) == 0 {
		t.Error("install spec must have packages, commands, or files")
	}

	if len(tl.Verify()) == 0 {
		t.Error("must have at least one verify command")
	}

	for _, dep := range tl.Dependencies() {
		if dep == tl.Name() {
			t.Errorf("tool depends on itself: %s", dep)
		}
	}
}

// AssertHasSkillMD checks that a tool's Skills() FS contains SKILL.md.
func AssertHasSkillMD(t *testing.T, tl tool.Tool) {
	t.Helper()
	s := tl.Skills()
	if s == nil {
		t.Fatal("Skills() returned nil")
	}
	_, err := fs.ReadFile(s, "SKILL.md")
	if err != nil {
		t.Errorf("expected SKILL.md in skills: %v", err)
	}
}
