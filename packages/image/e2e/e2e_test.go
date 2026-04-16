//go:build e2e

package e2e

import (
	"testing"

	runtimes "spwn.sh/catalog/runtimes"
	"spwn.sh/catalog/dependencies"
	ib "spwn.sh/packages/image"
	"spwn.sh/packages/image/imagetest"
)

func newRegistry(t *testing.T) *ib.Registry {
	t.Helper()
	reg := ib.NewRegistry()
	if err := dependencies.RegisterDefaults(reg); err != nil {
		t.Fatalf("register tools: %v", err)
	}
	if err := runtimes.RegisterDefaults(reg); err != nil {
		t.Fatalf("register runtimes: %v", err)
	}
	return reg
}

// ── Per-tool E2E tests ──

func TestUnix_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "@spwn/unix")

	imagetest.AssertBinaryExists(t, s, "bash")
	imagetest.AssertBinaryExists(t, s, "grep")
	imagetest.AssertBinaryExists(t, s, "sed")
	imagetest.AssertBinaryExists(t, s, "awk")
	imagetest.AssertBinaryExists(t, s, "curl")
	imagetest.AssertBinaryExists(t, s, "jq")
}

func TestGit_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "@spwn/unix", "@spwn/git")

	imagetest.AssertBinaryExists(t, s, "git")
	imagetest.AssertBinaryVersion(t, s, "git", "--version", "git version")
}

func TestNode_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "@spwn/unix", "@spwn/node")

	imagetest.AssertBinaryExists(t, s, "node")
	imagetest.AssertBinaryExists(t, s, "npm")
	imagetest.AssertBinaryExists(t, s, "npx")
	imagetest.AssertBinaryVersion(t, s, "node", "--version", "v20")
}

func TestPython_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "@spwn/unix", "@spwn/python")

	imagetest.AssertBinaryExists(t, s, "python3")
	imagetest.AssertBinaryVersion(t, s, "python3", "--version", "Python 3")
}

func TestClaudeCode_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "@spwn/unix", "@spwn/claude-code")

	imagetest.AssertBinaryExists(t, s, "claude")
	imagetest.AssertBinaryExists(t, s, "node") // transitive dep
	imagetest.AssertSkillInstalled(t, s, "@spwn/claude-code")
	imagetest.AssertFileExists(t, s, "/home/spwn/.claude.json")
	imagetest.AssertFileExists(t, s, "/home/spwn/.claude/settings.json")
}

func TestQmd_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "@spwn/unix", "@spwn/qmd")

	imagetest.AssertBinaryExists(t, s, "qmd")
	imagetest.AssertBinaryExists(t, s, "node") // transitive dep
	imagetest.AssertSkillInstalled(t, s, "@spwn/qmd")
	imagetest.AssertSkillContains(t, s, "@spwn/qmd", "QMD")
}

func TestCodex_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "@spwn/unix", "@spwn/codex")

	imagetest.AssertBinaryExists(t, s, "codex")
	imagetest.AssertBinaryExists(t, s, "node") // transitive dep
	imagetest.AssertSkillInstalled(t, s, "@spwn/codex")

	// Verify codex config was pre-configured
	imagetest.AssertFileExists(t, s, "/home/spwn/.codex/config.toml")
	imagetest.AssertFileContains(t, s, "/home/spwn/.codex/config.toml", "trust_level")
}

// ── Integration tests ──

func TestFullWorldStack_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t),
		"@spwn/unix", "@spwn/git", "@spwn/node", "@spwn/claude-code", "@spwn/cli", "@spwn/qmd",
	)

	binaries := []string{"bash", "grep", "curl", "git", "node", "npm", "claude", "qmd"}
	for _, bin := range binaries {
		imagetest.AssertBinaryExists(t, s, bin)
	}

	imagetest.AssertSkillInstalled(t, s, "@spwn/claude-code")
	imagetest.AssertSkillInstalled(t, s, "@spwn/cli")
	imagetest.AssertSkillInstalled(t, s, "@spwn/qmd")

	imagetest.AssertFileExists(t, s, "/world/skills/INDEX.md")
	imagetest.AssertFileContains(t, s, "/world/skills/INDEX.md", "claude-code")
	imagetest.AssertFileContains(t, s, "/world/skills/INDEX.md", "qmd")
}

func TestMinimalStack_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "@spwn/unix")

	imagetest.AssertBinaryExists(t, s, "bash")
	imagetest.AssertBinaryExists(t, s, "curl")

	// Node should NOT be present
	_, exitCode := s.Exec("command -v node")
	if exitCode == 0 {
		t.Error("node should not be present in minimal @spwn/unix stack")
	}
}

func TestDependencyAutoResolve_E2E(t *testing.T) {
	// Request @spwn/qmd without explicit @spwn/node - should auto-resolve
	s := imagetest.SpinUp(t, newRegistry(t), "@spwn/unix", "@spwn/qmd")

	imagetest.AssertBinaryExists(t, s, "qmd")
	imagetest.AssertBinaryExists(t, s, "node")
	imagetest.AssertBinaryVersion(t, s, "node", "--version", "v20")
}
