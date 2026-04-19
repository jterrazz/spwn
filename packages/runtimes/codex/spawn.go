package codex

import "spwn.sh/packages/runtimes"

// Spawner is the codex spawn-time adapter. Codex does not yet ship a
// BuildCommand (the architect invokes codex via ad-hoc bash in the
// architect daemon; no spwn-driven interactive codex sessions exist
// yet). The adapter is wired today so PrelaunchShell — the symlink
// that makes /credentials/openai/auth.json visible as ~/.codex/
// auth.json — lives next to the rest of codex's setup instead of
// leaking into architect/daemon.go.
var Spawner = &spawner{}

type spawner struct{}

// Name returns the runtime identifier.
func (*spawner) Name() string { return "codex" }

// BuildCommand returns an empty command. Codex is not yet wired as a
// primary runtime — sessions are launched via ad-hoc bash in the
// architect. Returning empty lets Spawner satisfy the interface
// without pretending to support something it doesn't.
func (*spawner) BuildCommand(runtimes.SpawnConfig) []string { return nil }

// SupportsSession reports whether codex can resume a prior session.
// Codex's session model maps to "thread_id" in its output, but spwn
// doesn't wire it as a resumable runtime yet.
func (*spawner) SupportsSession() bool { return false }

// Available gates the runtime behind feature-complete checks. Codex
// is available as an install target today; set true so consumers see
// it in the runtime list.
func (*spawner) Available() bool { return true }

// DefaultConfigFiles returns the files codex wants materialised into
// the agent's HOME at spawn time. Codex's config.toml is written at
// image-build time by Tool.Install (see tool.go UserCommands), so
// nothing extra is needed per-spawn today.
func (*spawner) DefaultConfigFiles(agentHome string) map[string][]byte { return nil }

// SyncHostCredentials is a no-op: codex's OAuth file lives at
// ~/.codex/auth.json on the host and is already picked up by
// packages/auth's provider resolver, which writes it to
// /credentials/openai/auth.json. No runtime-specific host sync is
// needed beyond that.
func (*spawner) SyncHostCredentials(credsDir string) error { return nil }

// PrelaunchShell returns the container-side shell fragment that
// wires /credentials/openai/auth.json into the location codex looks
// up on startup (~/.codex/auth.json). Runs as the agent user with
// /credentials bind-mounted read-only; guards with test-before-act so
// the launch never fails when OpenAI creds aren't configured.
//
// Intentionally omits `source /credentials/.env` — that belongs to the
// outer prelaunch composition, not this adapter. Callers that need
// env sourcing chain it themselves.
func (*spawner) PrelaunchShell() string {
	return `[ -f /credentials/openai/auth.json ] && mkdir -p $HOME/.codex && ln -sf /credentials/openai/auth.json $HOME/.codex/auth.json 2>/dev/null`
}
