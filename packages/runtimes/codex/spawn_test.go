package codex

import (
	"strings"
	"testing"

	"spwn.sh/packages/runtimes"
)

// TestSpawner_Name pins the runtime identifier. Renaming is a
// breaking change for every agent.yaml that uses `backend: codex`.
func TestSpawner_Name(t *testing.T) {
	if got := Spawner.Name(); got != "codex" {
		t.Errorf("Spawner.Name() = %q, want codex", got)
	}
}

// TestSpawner_PrelaunchShell locks in the codex OAuth plumbing: the
// shell snippet that symlinks /credentials/openai/auth.json to
// ~/.codex/auth.json whenever the file exists. This piece moved out
// of architect/daemon.go so codex auth lives with codex; a regression
// here silently breaks interactive codex sessions.
func TestSpawner_PrelaunchShell(t *testing.T) {
	got := Spawner.PrelaunchShell()
	for _, want := range []string{
		"/credentials/openai/auth.json", // source of the symlink
		"$HOME/.codex/auth.json",        // destination
		"ln -sf",                        // the symlink command
		"[ -f",                          // guard: source must exist
	} {
		if !strings.Contains(got, want) {
			t.Errorf("PrelaunchShell missing %q; got: %s", want, got)
		}
	}

	// The snippet must NOT source /credentials/.env — that's the
	// outer composer's job, not the adapter's. Including it here
	// would double-source when multiple adapters chain in daemon.go.
	if strings.Contains(got, "source /credentials/.env") {
		t.Errorf("PrelaunchShell must not source /credentials/.env (outer composer owns env loading); got: %s", got)
	}
}

// TestSpawner_SupportsSession reflects the current state: codex is
// not yet wired as a spwn-driven interactive runtime, so we advertise
// no session support. Flip when architect learns to drive codex
// sessions directly.
func TestSpawner_SupportsSession(t *testing.T) {
	if Spawner.SupportsSession() {
		t.Error("codex SupportsSession() should be false until spwn drives codex sessions")
	}
}

// TestSpawner_BuildCommand returns nil: codex sessions are launched
// via ad-hoc bash in the architect, not via a spwn-built command.
func TestSpawner_BuildCommand(t *testing.T) {
	if cmd := Spawner.BuildCommand(runtimes.SpawnConfig{}); cmd != nil {
		t.Errorf("BuildCommand should be nil until codex has a spwn-driven invocation; got %v", cmd)
	}
}

// TestSpawner_SyncHostCredentials is a no-op today: codex creds are
// resolved by packages/auth into /credentials/openai/auth.json,
// which the prelaunch shell then symlinks. Nothing runtime-specific
// to sync on the host.
func TestSpawner_SyncHostCredentials(t *testing.T) {
	if err := Spawner.SyncHostCredentials("/tmp/fake-creds"); err != nil {
		t.Errorf("SyncHostCredentials should be a no-op; got error: %v", err)
	}
}

// TestSpawner_DefaultConfigFiles returns nil: codex's config.toml is
// written at image-build time by Tool.Install UserCommands, not
// per-spawn. Keeps the spawn pipeline from re-seeding a file the
// runtime doesn't need.
func TestSpawner_DefaultConfigFiles(t *testing.T) {
	if got := Spawner.DefaultConfigFiles("/agents/neo"); got != nil {
		t.Errorf("DefaultConfigFiles should be nil; got %v", got)
	}
}

// TestAdapter pins the codex umbrella: install + spawn, no render.
func TestAdapter(t *testing.T) {
	if Adapter.Name != "codex" {
		t.Errorf("Adapter.Name = %q, want codex", Adapter.Name)
	}
	if Adapter.DefaultProvider != "openai" {
		t.Errorf("Adapter.DefaultProvider = %q, want openai", Adapter.DefaultProvider)
	}
	if Adapter.Tool == nil {
		t.Error("Adapter.Tool is nil — codex ships an install recipe")
	}
	if Adapter.Spawn == nil {
		t.Error("Adapter.Spawn is nil — codex ships prelaunch plumbing")
	}
	if Adapter.Render != nil {
		t.Error("Adapter.Render is non-nil — codex has no renderer today")
	}
}
