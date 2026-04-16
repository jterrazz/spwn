package imagetest

import (
	"spwn.sh/packages/dependency"
	"io/fs"
	"strings"
	"testing"

	ib "spwn.sh/packages/image"
)

// AssertValidTool checks all interface invariants on a tool.
func AssertValidTool(t *testing.T, tool ib.Tool) {
	t.Helper()

	if !strings.HasPrefix(tool.Name(), "@") {
		t.Errorf("name %q must start with @", tool.Name())
	}

	validKinds := map[dependency.Kind]bool{
		dependency.KindRuntime: true, dependency.KindTool: true,
		dependency.KindSDK: true, dependency.KindPlatform: true,
	}
	if !validKinds[tool.Kind()] {
		t.Errorf("invalid kind %q", tool.Kind())
	}

	if tool.Version() == "" {
		t.Error("version must not be empty")
	}

	spec := tool.Install()
	if len(spec.AptPackages) == 0 && len(spec.Commands) == 0 && len(spec.Files) == 0 {
		t.Error("install spec must have packages, commands, or files")
	}

	if len(tool.Verify()) == 0 {
		t.Error("must have at least one verify command")
	}

	for _, dep := range tool.Dependencies() {
		if dep == tool.Name() {
			t.Errorf("tool depends on itself: %s", dep)
		}
	}
}

// AssertHasSkillMD checks that a tool's Skills() FS contains SKILL.md.
func AssertHasSkillMD(t *testing.T, tool ib.Tool) {
	t.Helper()
	s := tool.Skills()
	if s == nil {
		t.Fatal("Skills() returned nil")
	}
	_, err := fs.ReadFile(s, "SKILL.md")
	if err != nil {
		t.Errorf("expected SKILL.md in skills: %v", err)
	}
}
