package architect

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/compile"
)

// TestMaterialiseWorldTree_SplitsByPrefix locks down the contract that
// world/* entries land on the host state directory and agents/* entries
// are docker-cp'd into the running container. Any other prefix is an
// error. This is the core of the docker-cp-not-bind-mount architecture.
func TestMaterialiseWorldTree_SplitsByPrefix(t *testing.T) {
	mb := newMockBackend()
	tree := compile.New()
	tree.AddString("world/physics.md", "laws of the world")
	tree.AddString("world/roster.md", "roster content")
	tree.AddString("agents/neo/CLAUDE.md", "neo entrypoint")
	tree.AddString("agents/neo/role.md", "worker")
	tree.AddString("agents/morpheus/CLAUDE.md", "morpheus entrypoint")

	stateDir := t.TempDir()
	err := materialiseWorldTree(context.Background(), mb, "ctr-1", tree, stateDir)
	if err != nil {
		t.Fatalf("materialiseWorldTree: %v", err)
	}

	// world/* went to the host state dir.
	for rel, want := range map[string]string{
		"physics.md": "laws of the world",
		"roster.md":  "roster content",
	} {
		b, err := os.ReadFile(filepath.Join(stateDir, rel))
		if err != nil {
			t.Fatalf("read host %s: %v", rel, err)
		}
		if string(b) != want {
			t.Errorf("world/%s content = %q, want %q", rel, b, want)
		}
	}

	// agents/* went via docker cp — one CopyTo per entry, using the
	// absolute container path.
	wantCp := map[string]string{
		"/agents/neo/CLAUDE.md":      "neo entrypoint",
		"/agents/neo/role.md":        "worker",
		"/agents/morpheus/CLAUDE.md": "morpheus entrypoint",
	}
	if len(mb.copyToCalls) != len(wantCp) {
		t.Fatalf("got %d CopyTo calls, want %d (calls: %+v)", len(mb.copyToCalls), len(wantCp), mb.copyToCalls)
	}
	for _, call := range mb.copyToCalls {
		if call.containerID != "ctr-1" {
			t.Errorf("CopyTo containerID = %q, want ctr-1", call.containerID)
		}
		want, ok := wantCp[call.destPath]
		if !ok {
			t.Errorf("unexpected CopyTo destPath %q", call.destPath)
			continue
		}
		if string(call.content) != want {
			t.Errorf("CopyTo %s content = %q, want %q", call.destPath, call.content, want)
		}
	}

	// No agent content leaked onto the host state dir.
	if _, err := os.Stat(filepath.Join(stateDir, "agents")); !os.IsNotExist(err) {
		t.Errorf("host state dir must not contain agents/ subtree, got err=%v", err)
	}
}

func TestMaterialiseWorldTree_UnknownPrefixIsError(t *testing.T) {
	mb := newMockBackend()
	tree := compile.New()
	tree.AddString("stray/foo.md", "should be rejected")

	err := materialiseWorldTree(context.Background(), mb, "ctr", tree, t.TempDir())
	if err == nil {
		t.Fatal("expected error for unknown prefix")
	}
}

// TestSyncAgentsInto_SkipsMissingHostDirs ensures that an agent whose
// host-side home does not exist does not fail the spawn — it's treated
// as a no-op so first-time scaffolds work.
func TestSyncAgentsInto_SkipsMissingHostDirs(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())
	mb := newMockBackend()

	err := syncAgentsInto(context.Background(), mb, "ctr", map[string]string{
		"ghost": "/agents/ghost",
	})
	if err != nil {
		t.Fatalf("syncAgentsInto: %v", err)
	}
	if len(mb.copyDirToCalls) != 0 {
		t.Errorf("expected zero CopyDirTo calls for missing host dir, got %+v", mb.copyDirToCalls)
	}
}

func TestSyncAgentsInto_CopiesPresentHostDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SPWN_HOME", home)
	// Create a fake host-side agent home under platform.AgentsDir().
	// platform.AgentsDir() reads from SPWN_HOME/agents by default.
	// We write a file so the Stat check passes.
	agentsRoot := filepath.Join(home, "agents")
	neoDir := filepath.Join(agentsRoot, "neo")
	if err := os.MkdirAll(neoDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	mb := newMockBackend()
	err := syncAgentsInto(context.Background(), mb, "ctr-abc", map[string]string{
		"neo": "/agents/neo",
	})
	if err != nil {
		t.Fatalf("syncAgentsInto: %v", err)
	}

	if len(mb.copyDirToCalls) != 1 {
		t.Fatalf("expected 1 CopyDirTo call, got %d: %+v", len(mb.copyDirToCalls), mb.copyDirToCalls)
	}
	call := mb.copyDirToCalls[0]
	if call.destDir != "/agents/neo" {
		t.Errorf("destDir = %q, want /agents/neo", call.destDir)
	}
	if call.hostSrcDir != neoDir {
		t.Errorf("hostSrcDir = %q, want %q", call.hostSrcDir, neoDir)
	}
}

// TestSyncAgentsOutOf_CopiesAllowlistedDirs verifies the sync-back
// contract: only journal/knowledge/playbooks/skills are pulled back,
// and failures are reported as warnings rather than aborting the loop.
func TestSyncAgentsOutOf_CopiesAllowlistedDirs(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())
	mb := newMockBackend()

	warnings := syncAgentsOutOf(context.Background(), mb, "ctr-xyz", map[string]string{
		"neo": "/agents/neo",
	})
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}

	wantSrcs := map[string]bool{
		"/agents/neo/journal":   true,
		"/agents/neo/knowledge": true,
		"/agents/neo/playbooks": true,
		"/agents/neo/skills":    true,
	}
	if len(mb.copyDirFromCalls) != len(wantSrcs) {
		t.Fatalf("got %d CopyDirFrom calls, want %d: %+v",
			len(mb.copyDirFromCalls), len(wantSrcs), mb.copyDirFromCalls)
	}
	for _, c := range mb.copyDirFromCalls {
		if !wantSrcs[c.srcDir] {
			t.Errorf("unexpected CopyDirFrom srcDir %q", c.srcDir)
		}
	}
}

func TestSyncAgentsOutOf_PerDirFailuresBecomeWarnings(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())
	mb := newMockBackend()
	mb.copyDirFromErrs = map[string]error{
		"/agents/neo/knowledge": fmt.Errorf("container has no /agents/neo/knowledge"),
	}

	warnings := syncAgentsOutOf(context.Background(), mb, "ctr", map[string]string{
		"neo": "/agents/neo",
	})

	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	// Other three dirs should still have been attempted.
	if len(mb.copyDirFromCalls) != 4 {
		t.Errorf("expected 4 CopyDirFrom attempts (one per allowlist entry), got %d",
			len(mb.copyDirFromCalls))
	}
}
