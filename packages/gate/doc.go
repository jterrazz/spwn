// Package gate is the spwn host-side credential broker and element
// bridge. It implements the gate concept from the spwn blueprint: a
// long-running container that holds credentials, hosts upstream MCP
// servers (or proxies to hosted ones), and exposes them to world
// containers as MCP-compliant HTTP endpoints.
//
// World containers never see credentials. They get tiny CLI wrappers
// (`notion-mcp`, `gmail-mcp`, …) that invoke `mcp2cli` against the
// gate's HTTP MCP server. The gate authenticates each upstream call
// using credentials from `~/.spwn/credentials/`, refreshes tokens
// proactively on its own schedule, and routes requests by element
// prefix (`/mcp/<element>/...`).
//
// # Architecture
//
//	World container                Gate (host-side)             Upstream
//	┌──────────────┐               ┌──────────────────┐         ┌─────────────────┐
//	│ notion-mcp   │ HTTP+MCP      │ /mcp/notion/* ──┼── HTTPS  │ mcp.notion.com  │
//	│  (mcp2cli    │──host.docker──│  + Authorization │  Bearer │  /mcp           │
//	│   wrapper)   │   .internal   │  Bearer token    │  token  │                 │
//	└──────────────┘  :9000        │ /mcp/gcp/*  ────┼── stdio  │ gws CLI         │
//	                               │  (gws-backed)    │         │  (gate-local)   │
//	                               └──────────────────┘         └─────────────────┘
//	                                       ▲
//	                                       │ reads
//	                                       │
//	                               ~/.spwn/credentials/
//
// # Public API
//
//	Server      — HTTP server hosting the element registry
//	Element     — interface: Name() + Handler() http.Handler
//	Registry    — keyed map of elements; serves the path multiplexer
//
// Element implementations (notion, gcp, …) live in this package as
// small files keyed by upstream provider.
package gate
