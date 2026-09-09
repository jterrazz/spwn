# Gate

> 🚧 **Experimental.** The gate container, the `gate:` block, the Node SDK, and the cookie-sync extension are in active development — schema, CLI, and behaviour will change without notice. Don't depend on it in production.

The gate is the one sanctioned crossing between the host and a world. Everything else about a world is closed by construction; the gate is where declared, scoped capability enters — a host service surfaced inside the world as an ordinary command, so the agent sees an element that exists and never an endpoint ([Physics](10-physics.md)).

Four concerns sit behind that: **capability bridging** (external services as plain CLI commands), **credential custody** (secrets and cookies stay host-side, never in a world's image or filesystem), **session relay** (operator traffic in, agent output back), and **supervision** (the bridged capability is spawned, health-checked, and respawned by the gate, not by the agent).

The shipped implementation is a long-running Docker container on the host (`spwn gate start`). It owns three concerns no individual world should:

1. **Cookie sync** — receives session cookies from the `spwn-cookie-sync` Chrome extension at `/sync/<provider>` and persists them under `~/.spwn/credentials/<provider>/cookies.json`.
2. **MCP routing** — exposes `/mcp/<element>/*` for every registered element (Notion proxy, Gmail/Gcal via `gws`, every catalog tool loaded from `~/.spwn/gate/tools/`).
3. **Browser primitive** — a Playwright Chromium sidecar (`apps/gate/browser/`, in-container `127.0.0.1:9001`) that catalog tools call to drive a cookie-loaded browser without shipping their own Chromium.

```
Host
└── spwn-gate container (port 9000 → host)
    ├── spwn-gate (Go)              ← cookie sync + MCP routing
    │     └── supervises:
    │         ├── gate-browser (Node, :9001)   ← Playwright pool
    │         └── catalog tools (Node, :9100+) ← per-tool MCP server
    ├── @spwn/gate-tool SDK          /usr/lib/node_modules/@spwn/gate-tool
    └── /gate/tools/<name>/          ← bind-mounted from ~/.spwn/gate/tools/
        └── tool.yaml + index.js
```

## Catalog tools that plug into the gate

A catalog entry under `catalog/<name>/tools/<name>/tool.yaml` becomes a gate element by adding a `gate:` section:

```yaml
name: "spwn:x"
gate:
  cookies:
    domains: [x.com, twitter.com]
    cookies: [auth_token, ct0]
  mcp:
    entry: ["node", "index.js", "mcp-serve"]
install:
  commands:
    - cat > /usr/local/bin/x-mcp <<'WRAPPER'
      #!/bin/bash
      spwn-policy-check x "${1:-}" || exit 1
      exec mcp2cli --mcp "http://host.docker.internal:9000/mcp/x" "$@"
      WRAPPER
```

At startup the gate scans `/gate/tools/`, auto-registers each tool's `CookieProvider` with cookie-sync (the extension picks it up next refresh), spawns its MCP subprocess on a port from `9100+`, and reverse-proxies `/mcp/<name>/*` into it. Adding a new site (LinkedIn, Reddit, …) is one new directory — no edits to `packages/gate/`.

## The Node SDK

Catalog tools `require('@spwn/gate-tool')` and use:

- `new Tool({ name }).method(name, { description, schema, handler })` — register MCP methods. The same `handler` serves both MCP calls (HTTP) and direct CLI invocation (`node index.js <method> --flags`).
- `openSession(provider)` — open a Playwright session in the sidecar with the provider's cookies pre-loaded. Returns a `Session` with `.goto / .click / .type / .scroll / .waitResponse / .eval / .end`.

Direct CLI mode is how host scripts (e.g. `publish.sh`) run writes without going through the agent's MCP wrapper — keeping human-in-the-loop methods out of agent reach by construction.

## Generic browser escape hatch

Beyond per-site catalog tools, the gate exposes the sidecar directly as `/mcp/browser` — agents that need ad-hoc browsing call `browser-open / browser-goto / browser-click / browser-eval / …` for sites without a dedicated tool. Heavier on tokens; reserve it for exploration, not scheduled scrapes.

## Per-agent allow/deny

Agents can constrain which methods of a dependency they may call:

```yaml
# agent.yaml
dependencies:
  - spwn:unix
  - name: spwn:x
    deny: [post-tweet, reply-tweet]   # read-only marketer
```

The compile pipeline materializes this as `/etc/spwn/policy/<short>.json` in the agent's image. The catalog tool's wrapper consults it via `spwn-policy-check <tool> <method>` (installed by `spwn:mcp2cli`) and rejects denied calls before they hit the gate. Merging is deny-takes-precedence when multiple agents in one world hold conflicting policies.

## Invariants

These hold whatever the implementation looks like:

1. **One crossing.** The gate connection is the only thing that crosses the container boundary; the Docker socket is never passed into a world.
2. **Bridges are declared, then verified.** A capability reaches a world only through its manifests, and only what was declared appears in the agent's faculties.
3. **Credentials never enter the world.** The wrapper carries the request; the gate holds the cookie or key and makes the authenticated call host-side.
4. **Scoping is per capability, not per service.** A bridge exposes named methods, and per-agent allow/deny keeps write-capable or human-in-the-loop methods out of agent reach by construction.
5. **Runtime-agnostic.** The bridge surfaces as ordinary commands, so any runtime that can run a shell can use it; swapping runtimes never touches the gate.

## Open question: communication between worlds

Undecided — whether two worlds should be able to reach each other. No direct communication (where things stand today) keeps isolation strong but rules out collaborative patterns; a host relay through the gate keeps isolation clean at the cost of latency and a host bottleneck; direct networking breaks the isolation model outright. Leaning toward the host relay.

## Related

- [Primitives](07-primitives.md) — the `gate:` block on a `tool.yaml`.
- [Physics](10-physics.md) — why a bridged capability is modelled as an element.
- [Architecture](01-architecture.md) — where the gate sits relative to worlds.
- [ADR-001](decisions/001-gate-over-socket.md) · [ADR-005](decisions/005-acp-inside-container.md) · [ADR-006](decisions/006-rust-container-gate.md) — the transport abstraction and the two-sided design that preceded the shipped host broker.
