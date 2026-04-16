package pack

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/deps"
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
	if err := os.MkdirAll(filepath.Join(agentDir, "identity"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"AGENTS.md", "agent.yaml"} {
		if err := os.WriteFile(filepath.Join(agentDir, f), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(agentDir, "identity", "profile.md"), []byte("# profile"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// withProject points platform.ProjectRoot at a scaffolded project and
// chdirs into it for the duration of a test. Needed because
// agent.AddPack / RemovePack resolve AgentDir via paths, and
// RunInstall walks up from cwd.
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

// ── ls ───────────────────────────────────────────────────────────────────────

func TestPackLs_empty(t *testing.T) {
	withProject(t)
	out, err := runWithOut(t, lsCmd)
	if err != nil {
		t.Fatalf("ls: %v", err)
	}
	if !strings.Contains(out.String(), "No packs installed") {
		t.Errorf("want empty message, got: %s", out.String())
	}
}

// ── install: ref-kind rejection ──────────────────────────────────────────────

func TestInstall_rejectsBareName(t *testing.T) {
	withProject(t)
	_, err := runWithOut(t, installCmd, "my-local-tool")
	if err == nil {
		t.Fatal("want error for bare name")
	}
	if !strings.Contains(err.Error(), "bare name") {
		t.Errorf("want bare-name hint, got: %v", err)
	}
}

func TestInstall_rejectsRegistryRef(t *testing.T) {
	withProject(t)
	_, err := runWithOut(t, installCmd, "@acme/foo")
	if err == nil {
		t.Fatal("want error for registry ref")
	}
	if !strings.Contains(err.Error(), "not yet supported") {
		t.Errorf("want registry-unsupported, got: %v", err)
	}
}

func TestInstall_rejectsUnknownBuiltin(t *testing.T) {
	withProject(t)
	SetCatalogLookup(func(pack string) bool { return false })
	t.Cleanup(func() { SetCatalogLookup(nil) })

	_, err := runWithOut(t, installCmd, "@spwn/nonesuch")
	if err == nil {
		t.Fatal("want error for unknown builtin")
	}
}

// ── install: happy path ──────────────────────────────────────────────────────

func TestInstall_addsToAgentAndLockfile(t *testing.T) {
	root := withProject(t)
	SetCatalogLookup(func(pack string) bool { return true })
	t.Cleanup(func() { SetCatalogLookup(nil) })

	if _, err := runWithOut(t, installCmd, "@spwn/git"); err != nil {
		t.Fatalf("install: %v", err)
	}

	lock, err := deps.LoadLockfile(root)
	if err != nil {
		t.Fatalf("load lockfile: %v", err)
	}
	if !lock.Has("@spwn/git") {
		t.Errorf("lockfile missing @spwn/git, got %+v", lock)
	}

	m, err := agent.LoadManifest("neo")
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	var found bool
	for _, p := range m.Deps {
		if p == "@spwn/git" {
			found = true
		}
	}
	if !found {
		t.Errorf("neo.agent.yaml missing @spwn/git, packages=%v", m.Deps)
	}
}

func TestInstall_idempotent(t *testing.T) {
	root := withProject(t)
	SetCatalogLookup(func(pack string) bool { return true })
	t.Cleanup(func() { SetCatalogLookup(nil) })

	for i := 0; i < 3; i++ {
		if _, err := runWithOut(t, installCmd, "@spwn/unix"); err != nil {
			t.Fatalf("install #%d: %v", i, err)
		}
	}
	m, _ := agent.LoadManifest("neo")
	count := 0
	for _, p := range m.Deps {
		if p == "@spwn/unix" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("want 1 instance of @spwn/unix, got %d", count)
	}
	lock, _ := deps.LoadLockfile(root)
	if !lock.Has("@spwn/unix") {
		t.Errorf("lockfile missing @spwn/unix")
	}
}

// ── uninstall ────────────────────────────────────────────────────────────────

func TestUninstall_removesEntry(t *testing.T) {
	root := withProject(t)
	SetCatalogLookup(func(pack string) bool { return true })
	t.Cleanup(func() { SetCatalogLookup(nil) })

	if _, err := runWithOut(t, installCmd, "@spwn/git"); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := runWithOut(t, uninstallCmd, "@spwn/git"); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	lock, _ := deps.LoadLockfile(root)
	if lock.Has("@spwn/git") {
		t.Errorf("lockfile still has @spwn/git after uninstall")
	}
	m, _ := agent.LoadManifest("neo")
	for _, p := range m.Deps {
		if p == "@spwn/git" {
			t.Errorf("neo.agent.yaml still has @spwn/git after uninstall")
		}
	}
}
