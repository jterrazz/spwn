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

// TestSpawner_SupportsSession pins codex's resumable-session
// contract. Codex sessions are identified by `thread_id`; the CLI
// resumes via `--thread <id>`. spwn plumbs the id through
// SpawnConfig.SessionID and BuildCommand emits the flag — see the
// oneshot_test.go suite for the detailed argv round-trip.
func TestSpawner_SupportsSession(t *testing.T) {
	if !Spawner.SupportsSession() {
		t.Error("codex SupportsSession() should be true now that BuildCommand wires --thread")
	}
}

// TestSpawner_BuildCommand_interactive returns `codex` plus the
// container-safety flags for interactive mode (no prompt, no named
// agent). The full one-shot / exec-subcommand argv is exercised in
// oneshot_test.go.
//
// Both safety flags MUST be on the interactive path too:
//   --skip-git-repo-check                       – /agents/<name> is not a git repo
//   --dangerously-bypass-approvals-and-sandbox  – bwrap can't nest inside Docker
//
// Regression guard: stripping either flag silently breaks `spwn agent
// talk` on a codex-backed world (first symptom: codex errors out with
// "Not inside a trusted directory" or "No permissions to create a
// new namespace").
func TestSpawner_BuildCommand_interactive(t *testing.T) {
	cmd := Spawner.BuildCommand(runtimes.SpawnConfig{})
	if len(cmd) == 0 || cmd[0] != "codex" {
		t.Errorf("interactive BuildCommand should start with `codex`; got %v", cmd)
	}
	joined := strings.Join(cmd, " ")
	for _, want := range []string{"--skip-git-repo-check", "--dangerously-bypass-approvals-and-sandbox"} {
		if !strings.Contains(joined, want) {
			t.Errorf("interactive BuildCommand missing %q; got %v", want, cmd)
		}
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

// TestSpawner_DefaultConfigFiles returns nil: the per-agent
// .codex/config.toml is emitted by the transpile renderer
// (GenerateAgentConfigTOML), and the project-trust entry is seeded
// by PrelaunchShell. No extra spawn-time file materialisation needed.
func TestSpawner_DefaultConfigFiles(t *testing.T) {
	if got := Spawner.DefaultConfigFiles("/agents/neo"); got != nil {
		t.Errorf("DefaultConfigFiles should be nil; got %v", got)
	}
}

// TestAdapter pins the codex umbrella as a full three-facet runtime:
// install + renderer + spawn-time plumbing. All three must stay
// registered for `spwn agent talk` on a codex-backed world to work
// end-to-end (same shape as claude-code).
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
	if Adapter.Render == nil {
		t.Error("Adapter.Render is nil — codex ships a source→Tree renderer")
	}
}
