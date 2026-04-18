package agent

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"spwn.sh/packages/agent"
)

// ── test helpers ────────────────────────────────────────────────────────────

func newComposeCmd() (*cobra.Command, *bytes.Buffer) {
	out := new(bytes.Buffer)
	cmd := &cobra.Command{Use: "test"}
	cmd.SetOut(out)
	cmd.SetErr(out)
	return cmd, out
}

func scaffoldAgent(t *testing.T, name string) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	agentDir := filepath.Join(tmp, "agents", name)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// SOUL.md at the agent root is what makes it a valid agent now —
	// the old identity/ directory was collapsed into a single file.
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("# soul\n"), 0o644); err != nil {
		t.Fatalf("write SOUL.md: %v", err)
	}
	return tmp
}

// resetComposeFlags clears the package-level compose flag state between
// runs. Cobra stores flag values in package-level vars, so tests that
// run sequentially leak state without this reset.
func resetComposeFlags() {
	composeDeps = nil
	composeSkills = nil
	composeTools = nil
	composeHooks = nil
	composeRemoves = nil
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
	composeDeps = []string{"spwn:python"}

	cmd, _ := newComposeCmd()
	err := addCmd.RunE(cmd, []string{"ghost"})
	if err == nil {
		t.Error("expected error when agent doesn't exist, got nil")
	}
}

func TestAgentAdd_SinglePackage(t *testing.T) {
	scaffoldAgent(t, "neo")
	resetComposeFlags()
	composeDeps = []string{"spwn:python"}

	cmd, _ := newComposeCmd()
	if err := addCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	m, err := agent.LoadManifest("neo")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(m.Deps) != 1 || m.Deps[0] != "spwn:python" {
		t.Errorf("Packages = %v, want [spwn:python]", m.Deps)
	}
}

func TestAgentAdd_MultiplePackages(t *testing.T) {
	scaffoldAgent(t, "neo")
	resetComposeFlags()
	composeDeps = []string{"spwn:unix", "spwn:python", "spwn:git"}

	cmd, _ := newComposeCmd()
	if err := addCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	m, _ := agent.LoadManifest("neo")
	if len(m.Deps) != 3 {
		t.Errorf("expected 3 packages, got %d: %v", len(m.Deps), m.Deps)
	}
}

func TestAgentAdd_Idempotent(t *testing.T) {
	scaffoldAgent(t, "neo")

	resetComposeFlags()
	composeDeps = []string{"spwn:python"}
	cmd, _ := newComposeCmd()
	if err := addCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatal(err)
	}

	resetComposeFlags()
	composeDeps = []string{"spwn:python"}
	cmd, _ = newComposeCmd()
	if err := addCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatal(err)
	}

	m, _ := agent.LoadManifest("neo")
	if len(m.Deps) != 1 {
		t.Errorf("expected 1 package (idempotent), got %d: %v", len(m.Deps), m.Deps)
	}
}

// TestAgentAdd_BareNameResolvesToCatalog verifies the CLI shorthand
// where `--dep qmd` auto-promotes to `spwn:qmd` via the catalog.
// The manifest stores the explicit scheme form — no bare names ever
// land on disk, even when the CLI accepts them.
func TestAgentAdd_BareNameResolvesToCatalog(t *testing.T) {
	scaffoldAgent(t, "neo")
	resetComposeFlags()
	composeDeps = []string{"python"}

	cmd, _ := newComposeCmd()
	if err := addCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	m, _ := agent.LoadManifest("neo")
	if len(m.Deps) != 1 || m.Deps[0] != "spwn:python" {
		t.Errorf("Packages = %v, want [spwn:python]", m.Deps)
	}
}

// TestAgentAdd_BareNameMissErrors verifies a bare name that doesn't
// match any catalog entry is rejected with a catalog-aware hint, and
// agent.yaml stays untouched (no half-written entries).
func TestAgentAdd_BareNameMissErrors(t *testing.T) {
	scaffoldAgent(t, "neo")
	resetComposeFlags()
	composeDeps = []string{"nonesuch"}

	cmd, _ := newComposeCmd()
	err := addCmd.RunE(cmd, []string{"neo"})
	if err == nil {
		t.Fatal("bare miss should error")
	}
	if !contains(err.Error(), "not in the catalog") {
		t.Errorf("error should mention 'not in the catalog': %v", err)
	}
	m, _ := agent.LoadManifest("neo")
	if len(m.Deps) != 0 {
		t.Errorf("manifest should stay empty on rejection, got %v", m.Deps)
	}
}

// TestAgentAdd_MixedFlags verifies all four flags (--dep / --skill /
// --tool / --hook) feed into the same deps list and share the same
// catalog resolver. The flag name is pure UX sugar; resolution is
// identical.
func TestAgentAdd_MixedFlags(t *testing.T) {
	scaffoldAgent(t, "neo")
	resetComposeFlags()
	composeDeps = []string{"python"}
	composeSkills = []string{"qmd"}
	composeTools = []string{"spwn:unix"}
	composeHooks = []string{"skill:focus"}

	cmd, _ := newComposeCmd()
	if err := addCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	m, _ := agent.LoadManifest("neo")
	got := map[string]bool{}
	for _, d := range m.Deps {
		got[d] = true
	}
	for _, want := range []string{"spwn:python", "spwn:qmd", "spwn:unix", "skill:focus"} {
		if !got[want] {
			t.Errorf("missing %q in %v", want, m.Deps)
		}
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

func TestAgentRemove_Package(t *testing.T) {
	scaffoldAgent(t, "neo")

	agent.AddDependency("neo", "spwn:unix")
	agent.AddDependency("neo", "spwn:python")

	resetComposeFlags()
	composeRemoves = []string{"spwn:unix"}
	cmd, _ := newComposeCmd()
	if err := removeCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatal(err)
	}

	m, _ := agent.LoadManifest("neo")
	if len(m.Deps) != 1 || m.Deps[0] != "spwn:python" {
		t.Errorf("Packages = %v, want [spwn:python]", m.Deps)
	}
}

func TestAgentRemove_AbsentPackageErrors(t *testing.T) {
	scaffoldAgent(t, "neo")
	agent.AddDependency("neo", "spwn:python")

	resetComposeFlags()
	composeRemoves = []string{"spwn:never-added"}
	cmd, _ := newComposeCmd()
	// Removing an absent package must error so scripts can distinguish
	// "I removed it" from "it was never there" — the previous
	// silent-success behaviour was QA finding #13.
	if err := removeCmd.RunE(cmd, []string{"neo"}); err == nil {
		t.Fatal("remove absent package should return an error, got nil")
	}

	// Manifest must stay untouched when the preflight rejects.
	m, _ := agent.LoadManifest("neo")
	if len(m.Deps) != 1 || m.Deps[0] != "spwn:python" {
		t.Errorf("Packages = %v, want [spwn:python] (unchanged)", m.Deps)
	}
}

// ── publish / pull stubs ─────────────────────────────────────────────────────

func TestAgentPublish_Stub(t *testing.T) {
	cmd, _ := newComposeCmd()
	err := publishCmd.RunE(cmd, []string{"neo"})
	if err == nil {
		t.Fatal("publish stub should return a not-implemented error")
	}
	var coder interface{ ExitCode() int }
	if !errors.As(err, &coder) || coder.ExitCode() != 2 {
		t.Errorf("expected ExitCode()==2, got err=%v", err)
	}
}

func TestAgentGet_Stub(t *testing.T) {
	cmd, _ := newComposeCmd()
	err := getCmd.RunE(cmd, []string{"@community/curie"})
	if err == nil {
		t.Fatal("get stub should return a not-implemented error")
	}
	var coder interface{ ExitCode() int }
	if !errors.As(err, &coder) || coder.ExitCode() != 2 {
		t.Errorf("expected ExitCode()==2, got err=%v", err)
	}
}

// ── misc ────────────────────────────────────────────────────────────────────

func contains(s, substr string) bool {
	return len(s) >= len(substr) && bytes.Contains([]byte(s), []byte(substr))
}
