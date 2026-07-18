# Gate

> 🚧 **Experimental.** The gate container, the `gate:` block, the Node SDK, and the cookie-sync extension are in active development — schema, CLI, and behaviour will change without notice. Don't depend on it in production.

The **gate** is a long-running Docker container on the host (`spwn gate start`). It owns three concerns no individual world should:

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

## Related

- [Primitives](04-primitives.md) — the `gate:` block on a `tool.yaml`.
- [Architecture](05-architecture.md) — where the gate sits relative to worlds.
