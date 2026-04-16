// Package architect is the world orchestrator. It sits on top of
// packages/world (state, runtime port, labels) and packages/image
// (backend, build) and turns a declarative spwn project into
// running Docker containers.
//
// The public entry points are:
//
//   - (*Architect).Spawn(ctx, opts) — build image, render tree,
//     start container, sync agents.
//   - (*Architect).Destroy(ctx, worldID) — stop + remove container,
//     sync memory back, persist state.
//   - StartDaemon / StopDaemon / GetDaemonStatus / TalkExecArgs —
//     lifecycle for the always-on `spwn-architect` container that
//     hosts the CLI inside Docker-over-Docker.
//
// Everything else in the package is helpers composed by Spawn:
// workspace resolution, image caching, deploy-tree materialise,
// credential injection.
package architect
