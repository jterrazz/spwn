//go:build e2e

package e2e

import (
	"testing"

	runtimes "spwn.sh/packages/runtimes"
	spwn "spwn.sh/packages/dependency/adapters/spwn"
	ib "spwn.sh/packages/compile"
	"spwn.sh/packages/compile/internal/imagetest"
)

func newRegistry(t *testing.T) *ib.Registry {
	t.Helper()
	reg := ib.NewRegistry()
	if err := spwn.RegisterDefaults(reg); err != nil {
		t.Fatalf("register tools: %v", err)
	}
	if err := runtimes.RegisterDefaults(reg); err != nil {
		t.Fatalf("register runtimes: %v", err)
	}
	return reg
}

// ── Per-tool E2E tests ──

func TestUnix_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "spwn:unix")

	imagetest.AssertBinaryExists(t, s, "bash")
	imagetest.AssertBinaryExists(t, s, "grep")
	imagetest.AssertBinaryExists(t, s, "sed")
	imagetest.AssertBinaryExists(t, s, "awk")
	imagetest.AssertBinaryExists(t, s, "curl")
	imagetest.AssertBinaryExists(t, s, "jq")
}

func TestGit_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "spwn:unix", "spwn:git")

	imagetest.AssertBinaryExists(t, s, "git")
	imagetest.AssertBinaryVersion(t, s, "git", "--version", "git version")
}

func TestNode_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "spwn:unix", "spwn:node")

	imagetest.AssertBinaryExists(t, s, "node")
	imagetest.AssertBinaryExists(t, s, "npm")
	imagetest.AssertBinaryExists(t, s, "npx")
	imagetest.AssertBinaryVersion(t, s, "node", "--version", "v20")
}

func TestPython_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "spwn:unix", "spwn:python")

	imagetest.AssertBinaryExists(t, s, "python3")
	imagetest.AssertBinaryVersion(t, s, "python3", "--version", "Python 3")
}

func TestClaudeCode_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "spwn:unix", "spwn:claude-code")

	// The native installer ships a self-contained claude binary — no
	// node, no SKILL.md (runtimes are transport, tools ship skills).
	imagetest.AssertBinaryExists(t, s, "claude")

	// ~/.claude.json and ~/.claude/settings.json are written at
	// spawn-time by runtime.DefaultConfigFiles, not at image-build
	// time. Image-level tests can't assert them.
}

func TestQmd_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "spwn:unix", "spwn:qmd")

	imagetest.AssertBinaryExists(t, s, "qmd")
	imagetest.AssertBinaryExists(t, s, "node") // transitive dep
	imagetest.AssertSkillInstalled(t, s, "spwn:qmd")
	imagetest.AssertSkillContains(t, s, "spwn:qmd", "QMD")
}

func TestCodex_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "spwn:unix", "spwn:codex")

	imagetest.AssertBinaryExists(t, s, "codex")
	imagetest.AssertBinaryExists(t, s, "node") // transitive dep
	// Runtimes don't ship a SKILL.md — only tools do (qmd, cli, …).

	// Verify codex config was pre-configured
	imagetest.AssertFileExists(t, s, "/home/spwn/.codex/config.toml")
	imagetest.AssertFileContains(t, s, "/home/spwn/.codex/config.toml", "trust_level")
}

// ── Integration tests ──

func TestFullWorldStack_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t),
		"spwn:unix", "spwn:git", "spwn:node", "spwn:claude-code", "spwn:cli", "spwn:qmd",
	)

	binaries := []string{"bash", "grep", "curl", "git", "node", "npm", "claude", "qmd"}
	for _, bin := range binaries {
		imagetest.AssertBinaryExists(t, s, bin)
	}

	// spwn:claude-code is a runtime (no SKILL.md); cli and qmd are
	// tools with skills.
	imagetest.AssertSkillInstalled(t, s, "spwn:cli")
	imagetest.AssertSkillInstalled(t, s, "spwn:qmd")

	imagetest.AssertFileExists(t, s, "/world/skills/INDEX.md")
	imagetest.AssertFileContains(t, s, "/world/skills/INDEX.md", "qmd")
}

func TestMinimalStack_E2E(t *testing.T) {
	s := imagetest.SpinUp(t, newRegistry(t), "spwn:unix")

	imagetest.AssertBinaryExists(t, s, "bash")
	imagetest.AssertBinaryExists(t, s, "curl")

	// Node should NOT be present
	_, exitCode := s.Exec("command -v node")
	if exitCode == 0 {
		t.Error("node should not be present in minimal spwn:unix stack")
	}
}

func TestDependencyAutoResolve_E2E(t *testing.T) {
	// Request spwn:qmd without explicit spwn:node - should auto-resolve
	s := imagetest.SpinUp(t, newRegistry(t), "spwn:unix", "spwn:qmd")

	imagetest.AssertBinaryExists(t, s, "qmd")
	imagetest.AssertBinaryExists(t, s, "node")
	imagetest.AssertBinaryVersion(t, s, "node", "--version", "v20")
}
