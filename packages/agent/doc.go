// Package agent owns the agent domain: composition manifests
// (agent.yaml), mind layers (playbooks, journal), per-world sessions,
// and evolution operations (dream, sleep, fork). Knowledge is NOT a
// Mind layer — it's world-scoped and lives at
// spwn/worlds/<name>/knowledge/ on the host (bind-mounted into
// /world/knowledge/ inside the container). Skills aren't a Mind
// layer either — they're build-time dependencies resolved via the
// `skill:` scheme or shipped by tools, and injected into
// /world/skills/ at image time.
//
// The public surface is intentionally narrow — most callers reach
// in through LoadManifest / SaveManifest / AddDependency /
// RemoveDependency for composition edits, InitMind / InspectAgent /
// ListAgents for lifecycle, and LoadSession / SaveSession for
// per-world conversation state. Evolution verbs (Dream, Sleep,
// Fork) are pure functions over the mind tree.
//
// Sub-packages (mind, journal, session, evolution) hold the
// implementation; the agent-level re-exports make the API
// consumable without reaching into them.
package agent
