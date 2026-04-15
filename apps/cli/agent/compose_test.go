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

	dir := filepath.Join(tmp, "agents", name, "identity")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return tmp
}

// resetComposeFlags clears the package-level compose flag state between
// runs. Cobra stores flag values in package-level vars, so tests that
// run sequentially leak state without this reset.
func resetComposeFlags() {
	composePlugins = nil
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
	composePlugins = []string{"@spwn/python"}

	cmd, _ := newComposeCmd()
	err := addCmd.RunE(cmd, []string{"ghost"})
	if err == nil {
		t.Error("expected error when agent doesn't exist, got nil")
	}
}

func TestAgentAdd_SinglePackage(t *testing.T) {
	scaffoldAgent(t, "neo")
	resetComposeFlags()
	composePlugins = []string{"@spwn/python"}

	cmd, _ := newComposeCmd()
	if err := addCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	m, err := agent.LoadManifest("neo")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(m.Plugins) != 1 || m.Plugins[0] != "@spwn/python" {
		t.Errorf("Packages = %v, want [@spwn/python]", m.Plugins)
	}
}

func TestAgentAdd_MultiplePackages(t *testing.T) {
	scaffoldAgent(t, "neo")
	resetComposeFlags()
	composePlugins = []string{"@spwn/unix", "@spwn/python", "@spwn/git"}

	cmd, _ := newComposeCmd()
	if err := addCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	m, _ := agent.LoadManifest("neo")
	if len(m.Plugins) != 3 {
		t.Errorf("expected 3 packages, got %d: %v", len(m.Plugins), m.Plugins)
	}
}

func TestAgentAdd_Idempotent(t *testing.T) {
	scaffoldAgent(t, "neo")

	resetComposeFlags()
	composePlugins = []string{"@spwn/python"}
	cmd, _ := newComposeCmd()
	if err := addCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatal(err)
	}

	resetComposeFlags()
	composePlugins = []string{"@spwn/python"}
	cmd, _ = newComposeCmd()
	if err := addCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatal(err)
	}

	m, _ := agent.LoadManifest("neo")
	if len(m.Plugins) != 1 {
		t.Errorf("expected 1 package (idempotent), got %d: %v", len(m.Plugins), m.Plugins)
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

	agent.AddPlugin("neo", "@spwn/unix")
	agent.AddPlugin("neo", "@spwn/python")

	resetComposeFlags()
	composePlugins = []string{"@spwn/unix"}
	cmd, _ := newComposeCmd()
	if err := removeCmd.RunE(cmd, []string{"neo"}); err != nil {
		t.Fatal(err)
	}

	m, _ := agent.LoadManifest("neo")
	if len(m.Plugins) != 1 || m.Plugins[0] != "@spwn/python" {
		t.Errorf("Packages = %v, want [@spwn/python]", m.Plugins)
	}
}

func TestAgentRemove_AbsentPackageErrors(t *testing.T) {
	scaffoldAgent(t, "neo")
	agent.AddPlugin("neo", "@spwn/python")

	resetComposeFlags()
	composePlugins = []string{"@spwn/never-added"}
	cmd, _ := newComposeCmd()
	// Removing an absent package must error so scripts can distinguish
	// "I removed it" from "it was never there" — the previous
	// silent-success behaviour was QA finding #13.
	if err := removeCmd.RunE(cmd, []string{"neo"}); err == nil {
		t.Fatal("remove absent package should return an error, got nil")
	}

	// Manifest must stay untouched when the preflight rejects.
	m, _ := agent.LoadManifest("neo")
	if len(m.Plugins) != 1 || m.Plugins[0] != "@spwn/python" {
		t.Errorf("Packages = %v, want [@spwn/python] (unchanged)", m.Plugins)
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
