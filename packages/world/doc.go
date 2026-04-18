// Package world is the world domain — the primitives that sit
// above the Docker daemon (via packages/container/backend) and below
// the orchestrator (packages/architect).
//
// The top-level package is a thin facade that re-exports the
// types consumers need: World, Workspace, Manifest, Status,
// AgentRecord, Store, Runtime, Backend. Implementation lives in
// sub-packages:
//
//   - manifest/      — spwn.yaml parse + scaffold defaults.
//   - models/        — World/Workspace/Status/AgentRecord types.
//   - state/         — ~/.spwn/state.json persistence.
//   - runtime/       — the Runtime adapter port interface.
//   - labels/        — Docker label-key constants (sh.spwn.*).
//   - runtimestate/  — container-state tracking via labels.
//
// External code (architect, apps/cli) imports the facade; sub-
// packages are consumed directly only when the caller genuinely
// needs the narrower surface (e.g. architect uses labels directly
// for orchestration).
package world
