package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"spwn.sh/packages/agent"
)

// ── test helpers ────────────────────────────────────────────────────────────

// newComposeCmd returns a cobra.Command set up with the persistent flags
// that newStepper() expects. It captures stdout/stderr into buffers so tests
// can assert on command output.
func newComposeCmd() (*cobra.Command, *bytes.Buffer) {
	out := new(bytes.Buffer)
	cmd := &cobra.Command{Use: "test"}
	cmd.SetOut(out)
	cmd.SetErr(out)
	return cmd, out
}

// scaffoldAgent creates a minimal agent directory that passes ValidateMind.
// Returns the temp SPWN_HOME so tests can inspect files under it.
func scaffoldAgent(t *testing.T, name string) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	dir := filepath.Join(tmp, "agents", name, "core")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return tmp
}

// resetComposeFlags clears the package-level compose flag state between runs.
// Cobra stores flag values in package-level vars, so tests that run
// sequentially leak state without this reset.
func resetComposeFlags() {
	composeTools = nil
	composeSkills = nil
	composeProfile = ""
	composeClearPro = false
}

// ── agent add ──────────────────────────────────────────────────────────────

func TestAgentAdd_NoFlagsReturnsError(t *testing.T) {
	scaffoldAgent(t, "neo")
	resetComposeFlags()

	cmd, _ := newComposeCmd()
	err := addCmd.RunE(cmd, []string{"neo"})
	if err == nil {
		t.Fatal("expected error when no flags provided, got nil")
	}
	if !contains(err.Error(), "nothing to add") {
		t.Errorf("error should mention 'nothing to add': %v", err)
	}
}

func TestAgentAdd_AgentNotFound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	resetComposeFlags()
	composeTools = []string{"@spwn/python"}

	cmd, _ := newComposeCmd()
	err := addCmd.RunE(cmd, []string{"ghost"})
	if err == nil {
		t.Error("expected error when agent doesn't exist, got nil")
	}
}

func TestAgentAdd_SingleTool(t *testing.T) {
	scaffoldAgent(t, "neo")
	resetComposeFlags()
	composeTools = []string{"@spwn/python"}

	cmd, _ := newComposeCmd()
	if err := addCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	m, err := agent.LoadManifest("neo")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(m.Tools) != 1 || m.Tools[0] != "@spwn/python" {
		t.Errorf("Tools = %v, want [@spwn/python]", m.Tools)
	}
}

func TestAgentAdd_MultipleToolsSkillsAndProfile(t *testing.T) {
	scaffoldAgent(t, "neo")
	resetComposeFlags()
	composeTools = []string{"@spwn/unix", "@spwn/python", "@spwn/git"}
	composeSkills = []string{"refactoring", "paper-reading"}
	composeProfile = "researcher"

	cmd, _ := newComposeCmd()
	if err := addCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	m, _ := agent.LoadManifest("neo")
	if len(m.Tools) != 3 {
		t.Errorf("expected 3 tools, got %d: %v", len(m.Tools), m.Tools)
	}
	if len(m.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d: %v", len(m.Skills), m.Skills)
	}
	if m.Profile != "researcher" {
		t.Errorf("Profile = %q, want \"researcher\"", m.Profile)
	}
}

func TestAgentAdd_Idempotent(t *testing.T) {
	scaffoldAgent(t, "neo")

	// First add.
	resetComposeFlags()
	composeTools = []string{"@spwn/python"}
	cmd, _ := newComposeCmd()
	if err := addCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatal(err)
	}

	// Second add with the same tool - should not duplicate.
	resetComposeFlags()
	composeTools = []string{"@spwn/python"}
	cmd, _ = newComposeCmd()
	if err := addCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatal(err)
	}

	m, _ := agent.LoadManifest("neo")
	if len(m.Tools) != 1 {
		t.Errorf("expected 1 tool (idempotent), got %d: %v", len(m.Tools), m.Tools)
	}
}

// ── agent remove ────────────────────────────────────────────────────────────

func TestAgentRemove_NoFlagsReturnsError(t *testing.T) {
	scaffoldAgent(t, "neo")
	resetComposeFlags()

	cmd, _ := newComposeCmd()
	err := removeCmd.RunE(cmd, []string{"neo"})
	if err == nil {
		t.Fatal("expected error when no flags provided, got nil")
	}
	if !contains(err.Error(), "nothing to remove") {
		t.Errorf("error should mention 'nothing to remove': %v", err)
	}
}

func TestAgentRemove_Tool(t *testing.T) {
	scaffoldAgent(t, "neo")

	// Seed with two tools.
	agent.AddTool("neo", "@spwn/unix")
	agent.AddTool("neo", "@spwn/python")

	// Remove one.
	resetComposeFlags()
	composeTools = []string{"@spwn/unix"}
	cmd, _ := newComposeCmd()
	if err := removeCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatal(err)
	}

	m, _ := agent.LoadManifest("neo")
	if len(m.Tools) != 1 || m.Tools[0] != "@spwn/python" {
		t.Errorf("Tools = %v, want [@spwn/python]", m.Tools)
	}
}

func TestAgentRemove_Skill(t *testing.T) {
	scaffoldAgent(t, "neo")
	agent.AddSkill("neo", "refactoring")
	agent.AddSkill("neo", "paper-reading")

	resetComposeFlags()
	composeSkills = []string{"paper-reading"}
	cmd, _ := newComposeCmd()
	if err := removeCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatal(err)
	}

	m, _ := agent.LoadManifest("neo")
	if len(m.Skills) != 1 || m.Skills[0] != "refactoring" {
		t.Errorf("Skills = %v, want [refactoring]", m.Skills)
	}
}

func TestAgentRemove_Profile(t *testing.T) {
	scaffoldAgent(t, "neo")
	agent.SetProfile("neo", "researcher")

	resetComposeFlags()
	composeClearPro = true
	cmd, _ := newComposeCmd()
	if err := removeCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatal(err)
	}

	m, _ := agent.LoadManifest("neo")
	if m.Profile != "" {
		t.Errorf("Profile = %q, want empty after clear", m.Profile)
	}
}

func TestAgentRemove_AbsentToolIsNoOp(t *testing.T) {
	scaffoldAgent(t, "neo")
	agent.AddTool("neo", "@spwn/python")

	resetComposeFlags()
	composeTools = []string{"@spwn/never-added"}
	cmd, _ := newComposeCmd()
	// Removing an absent block should NOT error.
	if err := removeCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Errorf("remove absent tool should be no-op, got: %v", err)
	}

	m, _ := agent.LoadManifest("neo")
	if len(m.Tools) != 1 {
		t.Errorf("Tools = %v, want unchanged", m.Tools)
	}
}

// ── publish / pull stubs ─────────────────────────────────────────────────────
//
// These are stubs until the registry ships - we just verify they don't
// blow up and print a "not yet implemented" placeholder.

func TestAgentPublish_Stub(t *testing.T) {
	cmd, out := newComposeCmd()
	if err := publishCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatalf("publish stub should not error: %v", err)
	}
	if !contains(out.String(), "not yet implemented") {
		t.Errorf("publish output should reference the placeholder: %s", out.String())
	}
}

func TestAgentGet_Stub(t *testing.T) {
	cmd, out := newComposeCmd()
	if err := getCmd.RunE(cmd, []string{"@community/curie"}); err != nil {
		t.Fatalf("get stub should not error: %v", err)
	}
	if !contains(out.String(), "not yet implemented") {
		t.Errorf("get output should reference the placeholder: %s", out.String())
	}
}

// ── misc ────────────────────────────────────────────────────────────────────

func contains(s, substr string) bool {
	return len(s) >= len(substr) && bytes.Contains([]byte(s), []byte(substr))
}
