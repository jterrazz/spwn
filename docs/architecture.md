# Architecture

Multi-module Go monorepo with Ports and Adapters architecture. 8 port interfaces, each with swappable adapters.

## Ports

| Port | What it abstracts | Default adapter |
|------|-------------------|-----------------|
| Runtime | How agents think | Claude Code (ACP) |
| Provider | Which LLM | Anthropic |
| Backend | Where worlds run | Docker |
| Channel | External communication | CLI |
| Memory | How profiles persist | Filesystem (markdown) |
| Store | How state is tracked | JSON file |
| Tool | What agents can do | Built-in + MCP |
| Skill | Reusable capabilities | Local files |

## Runtime adapters

Spwn treats agent runtimes as swappable adapters. The container-side Gate speaks [ACP](https://github.com/agentclientprotocol/agent-client-protocol), so adding a new runtime is a container image change.

| Runtime | Base Image | Status |
|---------|-----------|--------|
| Claude Code | node:20 | Available |
| Pi | node:20 | Available |
| Aider | python:3.12-slim | Available |
| Codex | node:20 | Planned |
| OpenCode | debian:bookworm-slim | Planned |
| Gemini | node:20 | Planned |

## Module map

```
spwn/
├── core/                       Domain libraries
│   ├── universe/                 World management, ports & adapters
│   ├── imagebuilder/             Composable Docker images, tool catalog
│   ├── agent/                    Agent lifecycle, profile, evolution
│   ├── gate/                     Host-container bridge
│   ├── messenger/                Inter-agent messaging
│   └── foundation/               Primitives (paths, IDs, constants)
├── examples/                   Bundled example gallery
├── apps/
│   ├── cli/                      The spwn binary
│   └── observatory/              Desktop app (Next.js + Tauri)
└── platform/
    ├── gate-runtime/             Container-side Rust gate
    └── fixtures/                 Test fixtures
```

## Roadmap

- ✅ World creation and isolation
- ✅ Persistent agent identity and memory
- ✅ Agent evolution (dream, sleep, forking)
- ✅ Multi-agent coordination and messaging
- ✅ Snapshots and rollback
- ✅ CLI and desktop app
- ✅ Pluggable runtime adapters (Claude Code, Pi, Aider)
- ✅ Activity log and audit trail
- ✅ Composable tool catalog with imagebuilder
- ⚪ Marketplace — share and import world templates, tool packs
- ⚪ Cloud-hosted worlds
- ⚪ Multi-universe federation
- ⚪ Mobile app
