//go:build e2e

// Package cli_test — knowledge mount E2E coverage for the
// worlds.<name>.knowledge refactor. Each test spawns a real container
// via the architect (against the existing spwn-test:latest mock
// image) and inspects the resulting binds / file tree to prove the
// "no knowledge = no mount + no skill reference" invariant.
//
// The tests reuse the same TestContext harness the packages/world
// e2e suite uses so they share isolation, cleanup, and Docker plumbing.
package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/architect"
	"spwn.sh/packages/world"
	"spwn.sh/packages/world/tests/e2e/setup"
)

// spawnForKnowledgeTest drives a single-agent spawn with an explicit
// knowledge path, letting each scenario toggle the bind cleanly.
// Returns the live AssertionChain plus the world record for
// downstream container inspection.
func spawnForKnowledgeTest(t *testing.T, tc *setup.TestContext, agentName, configName, knowledgePath string) *world.World {
	t.Helper()

	// Init the agent mind (the harness helper) so the architect's
	// mind-validation step passes.
	tc.InitAgent(agentName)

	opts := architect.SpawnOpts{
		ConfigName: configName,
		AgentName:  agentName,
		Image:      tc.Image,
		Knowledge:  knowledgePath,
	}

	result, err := tc.Arc.Spawn(context.Background(), opts)
	if err != nil {
		t.Fatalf("Spawn(%q): %v", configName, err)
	}
	tc.TrackWorld(result.World.ID)
	return result.World
}

// TestKnowledgeMountE1_UnsetMeansNoBind covers scenario E1:
// the world declares no knowledge path → no bind mount, AND every
// rendered file (AGENTS.md, mind-management skill, per-agent
// CLAUDE.md) omits every reference to /world/knowledge/.
func TestKnowledgeMountE1_UnsetMeansNoBind(t *testing.T) {
	// NewTestContext uses t.Setenv so t.Parallel would panic here.
	tc := setup.NewTestContext(t)

	w := spawnForKnowledgeTest(t, tc, "neo-e1", "no-knowledge-world", "")

	// The world state dir on the host must NOT contain a knowledge
	// sub-directory (the architect refactor dropped that default so
	// "no mount" really means "no /world/knowledge at all").
	hostStateDir := filepath.Join(tc.BaseDir, "worlds", w.ID)
	if _, err := os.Stat(filepath.Join(hostStateDir, "knowledge")); !os.IsNotExist(err) {
		t.Errorf("host world-state dir should NOT contain knowledge/; stat err=%v", err)
	}

	// Inside the container, /world/knowledge must NOT exist as a
	// bind mount. It MIGHT exist as an empty inherited dir from the
	// state dir — the spec accepts that case — but in our new model
	// we skip creating the knowledge subdir when no mount is set, so
	// a plain `test -d /world/knowledge` should fail.
	if tc.DirExistsInContainer(w.ContainerID, "/world/knowledge") {
		t.Errorf("/world/knowledge should not exist when no knowledge path is declared")
	}

	// Per-agent CLAUDE.md (which inlines every former /world/*.md
	// file) must not mention /world/knowledge/ when the mount is off.
	claudeMD := tc.ReadFileInContainer(w.ContainerID, "/agents/neo-e1/CLAUDE.md")
	if strings.Contains(claudeMD, "/world/knowledge/") {
		t.Errorf("per-agent CLAUDE.md should not mention /world/knowledge/:\n%s", claudeMD)
	}
	// The "knowledge base" language should also be absent.
	if strings.Contains(strings.ToLower(claudeMD), "knowledge base") {
		t.Errorf("per-agent CLAUDE.md should not mention a knowledge base:\n%s", claudeMD)
	}
}

// TestKnowledgeMountE2_SetMeansBindAndInjection covers scenario E2:
// the world declares a knowledge path AND the dir exists → bind
// mount flows through + the agent's system prompt is told about it.
func TestKnowledgeMountE2_SetMeansBindAndInjection(t *testing.T) {
	// NewTestContext uses t.Setenv so t.Parallel would panic here.
	tc := setup.NewTestContext(t)

	// Host-side knowledge dir with a known seed file so we can prove
	// the bind is live in both directions.
	knowledgeDir := filepath.Join(tc.BaseDir, "project-e2", "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatalf("mkdir knowledgeDir: %v", err)
	}
	seedPath := filepath.Join(knowledgeDir, "seed.md")
	if err := os.WriteFile(seedPath, []byte("HELLO-E2\n"), 0o644); err != nil {
		t.Fatalf("write seed.md: %v", err)
	}

	w := spawnForKnowledgeTest(t, tc, "neo-e2", "with-knowledge-world", knowledgeDir)

	// Inside the container, /world/knowledge/seed.md should appear
	// and carry the sentinel.
	if !tc.FileExistsInContainer(w.ContainerID, "/world/knowledge/seed.md") {
		t.Fatalf("/world/knowledge/seed.md missing inside container")
	}
	if got := strings.TrimSpace(tc.ReadFileInContainer(w.ContainerID, "/world/knowledge/seed.md")); got != "HELLO-E2" {
		t.Errorf("seed content mismatch: got %q, want %q", got, "HELLO-E2")
	}

	// Per-agent CLAUDE.md DOES mention /world/knowledge/ when mounted.
	claudeMD := tc.ReadFileInContainer(w.ContainerID, "/agents/neo-e2/CLAUDE.md")
	if !strings.Contains(claudeMD, "/world/knowledge/") {
		t.Errorf("per-agent CLAUDE.md should mention /world/knowledge/ when mounted:\n%s", claudeMD)
	}

	// Bind mount roundtrip: writing inside the container should
	// appear on the host side.
	tc.ExecInContainer(w.ContainerID, []string{"sh", "-c", "echo FROM-INSIDE > /world/knowledge/from-inside.md"})
	written, err := os.ReadFile(filepath.Join(knowledgeDir, "from-inside.md"))
	if err != nil {
		t.Fatalf("host-side roundtrip read: %v", err)
	}
	if strings.TrimSpace(string(written)) != "FROM-INSIDE" {
		t.Errorf("roundtrip content mismatch: %q", string(written))
	}
}

// TestKnowledgeMountE3_MultipleWorldsNoCrossPollution covers E3:
// two worlds, two distinct knowledge paths, zero leakage between
// them. Each world sees only its own seed file.
func TestKnowledgeMountE3_MultipleWorldsNoCrossPollution(t *testing.T) {
	// NewTestContext uses t.Setenv so t.Parallel would panic here.
	tc := setup.NewTestContext(t)

	alphaDir := filepath.Join(tc.BaseDir, "project-e3", "knowledge-alpha")
	betaDir := filepath.Join(tc.BaseDir, "project-e3", "knowledge-beta")
	for _, d := range []string{alphaDir, betaDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	if err := os.WriteFile(filepath.Join(alphaDir, "alpha-only.md"), []byte("A\n"), 0o644); err != nil {
		t.Fatalf("write alpha seed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(betaDir, "beta-only.md"), []byte("B\n"), 0o644); err != nil {
		t.Fatalf("write beta seed: %v", err)
	}

	alpha := spawnForKnowledgeTest(t, tc, "neo-e3-alpha", "alpha-world", alphaDir)
	beta := spawnForKnowledgeTest(t, tc, "neo-e3-beta", "beta-world", betaDir)

	// Alpha sees only its own file.
	if !tc.FileExistsInContainer(alpha.ContainerID, "/world/knowledge/alpha-only.md") {
		t.Errorf("alpha: /world/knowledge/alpha-only.md missing")
	}
	if tc.FileExistsInContainer(alpha.ContainerID, "/world/knowledge/beta-only.md") {
		t.Errorf("alpha: /world/knowledge/beta-only.md leaked from beta")
	}
	// Beta sees only its own file.
	if !tc.FileExistsInContainer(beta.ContainerID, "/world/knowledge/beta-only.md") {
		t.Errorf("beta: /world/knowledge/beta-only.md missing")
	}
	if tc.FileExistsInContainer(beta.ContainerID, "/world/knowledge/alpha-only.md") {
		t.Errorf("beta: /world/knowledge/alpha-only.md leaked from alpha")
	}
}

// TestKnowledgeMountE4_MixedWorldsNoLeakage covers E4: one world has
// knowledge, one doesn't. The with-knowledge world has both the
// mount AND the skill text; the no-knowledge world has neither.
func TestKnowledgeMountE4_MixedWorldsNoLeakage(t *testing.T) {
	// NewTestContext uses t.Setenv so t.Parallel would panic here.
	tc := setup.NewTestContext(t)

	alphaDir := filepath.Join(tc.BaseDir, "project-e4", "knowledge")
	if err := os.MkdirAll(alphaDir, 0o755); err != nil {
		t.Fatalf("mkdir alphaDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(alphaDir, "alpha-seed.md"), []byte("Z\n"), 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}

	alpha := spawnForKnowledgeTest(t, tc, "neo-e4-alpha", "alpha", alphaDir)
	beta := spawnForKnowledgeTest(t, tc, "neo-e4-beta", "beta", "")

	// Alpha: mount + CLAUDE.md injection.
	if !tc.FileExistsInContainer(alpha.ContainerID, "/world/knowledge/alpha-seed.md") {
		t.Errorf("alpha: seed missing from /world/knowledge")
	}
	alphaClaude := tc.ReadFileInContainer(alpha.ContainerID, "/agents/neo-alpha/CLAUDE.md")
	if !strings.Contains(alphaClaude, "/world/knowledge/") {
		t.Errorf("alpha: CLAUDE.md should mention /world/knowledge/")
	}

	// Beta: no mount, no mention of /world/knowledge/ anywhere.
	if tc.DirExistsInContainer(beta.ContainerID, "/world/knowledge") {
		t.Errorf("beta: /world/knowledge should not exist")
	}
	betaClaude := tc.ReadFileInContainer(beta.ContainerID, "/agents/neo-beta/CLAUDE.md")
	if strings.Contains(betaClaude, "/world/knowledge/") {
		t.Errorf("beta: CLAUDE.md should not mention /world/knowledge/:\n%s", betaClaude)
	}
}
