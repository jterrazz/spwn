package architect

import (
	"fmt"

	"spwn.sh/packages/runtimes"
	"spwn.sh/packages/world/models"
)

// defaultRuntimeName is the runtime the architect falls back to when
// a world record is missing its Runtime field. That happens for
// worlds spawned before models.World.Runtime was persisted, and for
// any caller that leaves SpawnOpts.RuntimeName empty.
const defaultRuntimeName = "claude-code"

// resolveRuntimeName returns the runtime name associated with a
// world record, with the legacy fallback applied. Pure string math —
// no registry lookup — so callers that need to pass a name to
// transpile.Compile (which takes a name, not a spawner) can use
// this directly.
func resolveRuntimeName(u *models.World) string {
	if u != nil && u.Runtime != "" {
		return u.Runtime
	}
	return defaultRuntimeName
}

// resolveSpawner returns the runtimes.Spawner whose adapter is
// registered under the world's declared runtime (or the default
// when the field is empty / the record is nil).
//
// This is the per-world routing hook: a world spawned under
// `runtime.backend: spwn:codex` will resolve to codex's spawner
// even when the architect was constructed without knowing about
// codex. Every interactive operation — SpawnAgent, SpawnAgentDetached,
// SpawnNPC, hot-deploy — routes through here so the correct
// runtime's BuildCommand / PrelaunchShell is used.
//
// Callers must always pass the live world record (from
// rstate.Get) — not the SpawnOpts at construction time — so that a
// world's chosen runtime is pinned for its whole lifetime.
func (a *Architect) resolveSpawner(u *models.World) (runtimes.Spawner, error) {
	name := resolveRuntimeName(u)
	rt, err := runtimes.GetSpawner(name)
	if err != nil || rt == nil {
		return nil, fmt.Errorf("world runtime %q is not registered: %w", name, err)
	}
	return rt, nil
}
