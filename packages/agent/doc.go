// Package agent owns the agent domain: composition manifests
// (agent.yaml), mind layers (identity, skills, knowledge, playbooks,
// journal), per-world sessions, and evolution operations (dream,
// sleep, fork).
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
