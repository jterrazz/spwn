# Observatory

The Observatory is the web face of spwn: every active world and agent at a glance, chat with them, and lifecycle actions from the interface. Where the CLI is the scripting surface, this is the monitoring and interactive one. Open it with `spwn web`.

## Shape

A Next.js application ([`apps/web`](../apps/web)) backed by a Go API server ([`apps/api`](../apps/api)) that reads and writes the spwn filesystem directly and calls the same domain packages the CLI consumes, behind HTTP. Where those two sit in the layer graph is [Architecture](01-architecture.md).

## Design principles

- **Read-write, not read-only.** Full CRUD over worlds and agents — spawn, snapshot, destroy, browse and edit a Mind — not a metrics viewer.
- **The filesystem is the truth.** Views are computed live from the same on-disk state the CLI uses: no shadow database, no background workers.
- **CLI parity.** Every action available in the CLI is reachable from the dashboard; the CLI stays the primary surface for scripting and automation.
- **Zero friction.** Dependencies that are not running are started when needed, and failures come with an actionable hint.

## Designed, not shipped

The long-term target is an isometric, real-time visualisation of worlds and the agents working inside them, fed by runtime event streaming ([ADR-008](decisions/008-rivet-runtime-layer.md)). Today the dashboard is the flat web UI described above.

## Related

- [CLI](06-cli.md) — the other operator surface.
- [Architecture](01-architecture.md) — `apps/web` and `apps/api` in the monorepo layout.
