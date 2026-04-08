//go:build e2e

package e2e

import (
	"testing"

	ib "spwn.sh/core/imagebuilder"
	"spwn.sh/core/imagebuilder/catalog"
	"spwn.sh/core/imagebuilder/imagebuildertest"
)

func newRegistry(t *testing.T) *ib.Registry {
	t.Helper()
	reg := ib.NewRegistry()
	catalog.RegisterDefaults(reg)
	return reg
}

// ── Per-tool E2E tests ──

func TestUnix_E2E(t *testing.T) {
	s := imagebuildertest.SpinUp(t, newRegistry(t), "@unix")

	imagebuildertest.AssertBinaryExists(t, s, "bash")
	imagebuildertest.AssertBinaryExists(t, s, "grep")
	imagebuildertest.AssertBinaryExists(t, s, "sed")
	imagebuildertest.AssertBinaryExists(t, s, "awk")
	imagebuildertest.AssertBinaryExists(t, s, "curl")
	imagebuildertest.AssertBinaryExists(t, s, "jq")
}

func TestGit_E2E(t *testing.T) {
	s := imagebuildertest.SpinUp(t, newRegistry(t), "@unix", "@git")

	imagebuildertest.AssertBinaryExists(t, s, "git")
	imagebuildertest.AssertBinaryVersion(t, s, "git", "--version", "git version")
}

func TestNode_E2E(t *testing.T) {
	s := imagebuildertest.SpinUp(t, newRegistry(t), "@unix", "@node")

	imagebuildertest.AssertBinaryExists(t, s, "node")
	imagebuildertest.AssertBinaryExists(t, s, "npm")
	imagebuildertest.AssertBinaryExists(t, s, "npx")
	imagebuildertest.AssertBinaryVersion(t, s, "node", "--version", "v20")
}

func TestPython_E2E(t *testing.T) {
	s := imagebuildertest.SpinUp(t, newRegistry(t), "@unix", "@python")

	imagebuildertest.AssertBinaryExists(t, s, "python3")
	imagebuildertest.AssertBinaryVersion(t, s, "python3", "--version", "Python 3")
}

func TestClaudeCode_E2E(t *testing.T) {
	s := imagebuildertest.SpinUp(t, newRegistry(t), "@unix", "@claude-code")

	imagebuildertest.AssertBinaryExists(t, s, "claude")
	imagebuildertest.AssertBinaryExists(t, s, "node") // transitive dep
	imagebuildertest.AssertSkillInstalled(t, s, "@claude-code")
	imagebuildertest.AssertFileExists(t, s, "/home/spwn/.claude.json")
	imagebuildertest.AssertFileExists(t, s, "/home/spwn/.claude/settings.json")
}

func TestQmd_E2E(t *testing.T) {
	s := imagebuildertest.SpinUp(t, newRegistry(t), "@unix", "@qmd")

	imagebuildertest.AssertBinaryExists(t, s, "qmd")
	imagebuildertest.AssertBinaryExists(t, s, "node") // transitive dep
	imagebuildertest.AssertSkillInstalled(t, s, "@qmd")
	imagebuildertest.AssertSkillContains(t, s, "@qmd", "QMD")
}

// ── Integration tests ──

func TestFullWorldStack_E2E(t *testing.T) {
	s := imagebuildertest.SpinUp(t, newRegistry(t),
		"@unix", "@git", "@node", "@claude-code", "@spwn", "@qmd",
	)

	binaries := []string{"bash", "grep", "curl", "git", "node", "npm", "claude", "qmd"}
	for _, bin := range binaries {
		imagebuildertest.AssertBinaryExists(t, s, bin)
	}

	imagebuildertest.AssertSkillInstalled(t, s, "@claude-code")
	imagebuildertest.AssertSkillInstalled(t, s, "@spwn")
	imagebuildertest.AssertSkillInstalled(t, s, "@qmd")

	imagebuildertest.AssertFileExists(t, s, "/world/skills/INDEX.md")
	imagebuildertest.AssertFileContains(t, s, "/world/skills/INDEX.md", "claude-code")
	imagebuildertest.AssertFileContains(t, s, "/world/skills/INDEX.md", "qmd")
}

func TestMinimalStack_E2E(t *testing.T) {
	s := imagebuildertest.SpinUp(t, newRegistry(t), "@unix")

	imagebuildertest.AssertBinaryExists(t, s, "bash")
	imagebuildertest.AssertBinaryExists(t, s, "curl")

	// Node should NOT be present
	_, exitCode := s.Exec("command -v node")
	if exitCode == 0 {
		t.Error("node should not be present in minimal @unix stack")
	}
}

func TestDependencyAutoResolve_E2E(t *testing.T) {
	// Request @qmd without explicit @node — should auto-resolve
	s := imagebuildertest.SpinUp(t, newRegistry(t), "@unix", "@qmd")

	imagebuildertest.AssertBinaryExists(t, s, "qmd")
	imagebuildertest.AssertBinaryExists(t, s, "node")
	imagebuildertest.AssertBinaryVersion(t, s, "node", "--version", "v20")
}
