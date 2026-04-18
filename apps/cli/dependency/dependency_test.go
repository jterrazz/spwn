package dependency

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/dependency"
)

func runWithOut(t *testing.T, c *cobra.Command, args ...string) (*bytes.Buffer, error) {
	t.Helper()
	out := new(bytes.Buffer)
	cmd := &cobra.Command{Use: "test"}
	cmd.SetOut(out)
	cmd.SetErr(out)
	return out, c.RunE(cmd, args)
}

func scaffoldProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "spwn.yaml"), []byte(`version: 2
name: test-proj
worlds:
  default:
    agents: [neo]
    workspaces: [.]
`), 0o644); err != nil {
		t.Fatal(err)
	}
	agentDir := filepath.Join(root, "spwn", "agents", "neo")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"AGENTS.md", "agent.yaml"} {
		if err := os.WriteFile(filepath.Join(agentDir, f), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// SOUL.md at the agent root replaces the old identity/profile.md.
	if err := os.WriteFile(filepath.Join(agentDir, "SOUL.md"), []byte("# soul\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// withProject points platform.ProjectRoot at a scaffolded project and
// chdirs into it for the duration of a test. Needed because
// agent.AddDependency / RemoveDependency resolve AgentDir via paths,
// and RunInstall walks up from cwd.
func withProject(t *testing.T) string {
	t.Helper()
	root := scaffoldProject(t)
	oldCwd, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	platform.SetProjectRoot(root)
	t.Cleanup(func() {
		platform.SetProjectRoot("")
		os.Chdir(oldCwd)
	})
	return root
}

// ── install: ref-kind rejection ──────────────────────────────────────────────

func TestInstall_rejectsBareName(t *testing.T) {
	withProject(t)
	// No catalog wired, so the bare-name resolver fails with the
	// "not in the catalog" hint. This is the happy path: the CLI
	// shorthand tried to auto-promote `my-local-tool` to
	// `spwn:my-local-tool` and missed, and the error surfaces the
	// local-scheme alternative so the user can correct.
	_, err := runWithOut(t, installCmd, "my-local-tool")
	if err == nil {
		t.Fatal("want error for bare name")
	}
	if !strings.Contains(err.Error(), "not in the catalog") {
		t.Errorf("want catalog-miss hint, got: %v", err)
	}
	if !strings.Contains(err.Error(), "skill:my-local-tool") {
		t.Errorf("error should suggest the local-scheme alternative: %v", err)
	}
}

func TestInstall_rejectsLocalSchemeRef(t *testing.T) {
	withProject(t)
	// Without --agent, installing a local ref is refused because
	// Bolting a local block onto every agent is almost never what
	// The user wants. The error points at --agent as the fix.
	_, err := runWithOut(t, installCmd, "skill:paper-reading")
	if err == nil {
		t.Fatal("want error for local ref without --agent")
	}
	if !strings.Contains(err.Error(), "--agent") {
		t.Errorf("want --agent hint, got: %v", err)
	}
}

func TestInstall_rejectsRegistryRef(t *testing.T) {
	withProject(t)
	_, err := runWithOut(t, installCmd, "github:acme/foo")
	if err == nil {
		t.Fatal("want error for registry ref")
	}
	if !strings.Contains(err.Error(), "not yet supported") {
		t.Errorf("want registry-unsupported, got: %v", err)
	}
}

func TestInstall_rejectsLegacyAtRef(t *testing.T) {
	withProject(t)
	_, err := runWithOut(t, installCmd, "@acme/foo")
	if err == nil {
		t.Fatal("want error for legacy @ ref")
	}
	// Legacy @owner/name fails the grammar (it's neither a bare
	// identifier nor an explicit scheme), so the resolver surfaces
	// the "malformed" hint pointing at the five valid schemes.
	if !strings.Contains(err.Error(), "malformed") {
		t.Errorf("want malformed-grammar hint, got: %v", err)
	}
}

func TestInstall_rejectsUnknownBuiltin(t *testing.T) {
	withProject(t)
	SetCatalogLookup(func(ref string) bool { return false }, nil)
	t.Cleanup(func() { SetCatalogLookup(nil, nil) })

	_, err := runWithOut(t, installCmd, "spwn:nonesuch")
	if err == nil {
		t.Fatal("want error for unknown builtin")
	}
}

// ── install: happy path ──────────────────────────────────────────────────────

func TestInstall_addsToAgentAndLockfile(t *testing.T) {
	root := withProject(t)
	SetCatalogLookup(func(ref string) bool { return true }, nil)
	t.Cleanup(func() { SetCatalogLookup(nil, nil) })

	if _, err := runWithOut(t, installCmd, "spwn:git"); err != nil {
		t.Fatalf("install: %v", err)
	}

	lock, err := dependency.LoadLockfile(root)
	if err != nil {
		t.Fatalf("load lockfile: %v", err)
	}
	if !lock.Has("spwn:git") {
		t.Errorf("lockfile missing spwn:git, got %+v", lock)
	}

	m, err := agent.LoadManifest("neo")
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	var found bool
	for _, p := range m.Deps {
		if p == "spwn:git" {
			found = true
		}
	}
	if !found {
		t.Errorf("neo.agent.yaml missing spwn:git, deps=%v", m.Deps)
	}
}

func TestInstall_idempotent(t *testing.T) {
	root := withProject(t)
	SetCatalogLookup(func(ref string) bool { return true }, nil)
	t.Cleanup(func() { SetCatalogLookup(nil, nil) })

	for i := 0; i < 3; i++ {
		if _, err := runWithOut(t, installCmd, "spwn:unix"); err != nil {
			t.Fatalf("install #%d: %v", i, err)
		}
	}
	m, _ := agent.LoadManifest("neo")
	count := 0
	for _, p := range m.Deps {
		if p == "spwn:unix" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("want 1 instance of spwn:unix, got %d", count)
	}
	lock, _ := dependency.LoadLockfile(root)
	if !lock.Has("spwn:unix") {
		t.Errorf("lockfile missing spwn:unix")
	}
}

// ── uninstall ────────────────────────────────────────────────────────────────

func TestUninstall_removesEntry(t *testing.T) {
	root := withProject(t)
	SetCatalogLookup(func(ref string) bool { return true }, nil)
	t.Cleanup(func() { SetCatalogLookup(nil, nil) })

	if _, err := runWithOut(t, installCmd, "spwn:git"); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := runWithOut(t, uninstallCmd, "spwn:git"); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	lock, _ := dependency.LoadLockfile(root)
	if lock.Has("spwn:git") {
		t.Errorf("lockfile still has spwn:git after uninstall")
	}
	m, _ := agent.LoadManifest("neo")
	for _, p := range m.Deps {
		if p == "spwn:git" {
			t.Errorf("neo.agent.yaml still has spwn:git after uninstall")
		}
	}
}
