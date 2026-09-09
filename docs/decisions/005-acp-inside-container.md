# ADR-005: Agent Client Protocol Inside the Container

**Date:** 2026-03-01
**Status:** Accepted

## Context

Spwn needs a standardized way to communicate with the agent CLI running inside a container. The current design has the Architect spawning Claude Code CLI directly inside the container and managing it via process signals and `--resume` flags. This works but is tightly coupled to Claude Code's specific CLI interface.

The [Agent Client Protocol (ACP)](https://github.com/agentclientprotocol/agent-client-protocol) is a standardized JSON-RPC protocol (over stdio) between clients and coding agents. It supports session management, authentication, streaming, and tool permissions. 34+ agents support it, including Claude Code, Codex CLI, and Gemini CLI.

The question: should Spwn build its own agent communication protocol, adopt ACP everywhere, or use ACP selectively?

## Decision

Run an ACP client **inside** the container, alongside the agent CLI. The Gate bridges between the container-side ACP client and the host-side Architect over TCP (`host.docker.internal`). Spwn only builds the Gate bridge protocol—not the agent communication protocol.

The architecture:

```
Host                           │  Universe (container)
                                    │
Architect                           │
  └── Gate (host-side)              │  Gate (container-side)
       │                            │    │
       └────── TCP (host.docker) ───│────┘
                                    │    └── ACP Client
                                    │         └── stdio → Agent CLI
```

1. **ACP client** runs inside the container—speaks stdio to Claude Code CLI (or any ACP-compatible agent)
2. **Gate (container-side)** wraps the ACP client, handles local agent lifecycle
3. **Gate (host-side)** runs on the Host, bridges requests from the Architect
4. **TCP** connects the two Gate halves—the only thing that crosses the container boundary
5. **Architect** speaks Spwn's own Gate protocol to the host-side Gate—never sees ACP directly

### What ACP Gives Us for Free

- **Session management**—`session/new`, `session/load`, `session/prompt`, resume across restarts
- **Authentication**—`authenticate` flow handles API key setup
- **Streaming**—`session/update` events for real-time agent output
- **Tool permissions**—control what the agent can do programmatically
- **Multi-agent support**—swap the agent CLI (Claude Code → Codex CLI → Gemini CLI) without changing the Gate protocol

### What Spwn Builds

- **Gate bridge protocol**—the wire format over TCP between host and container
- **Element bridging**—MCP servers on the Host exposed as CLI commands (unchanged)
- **Mount management**—Mind, workspace mounts (unchanged)
- **Lifecycle orchestration**—the Architect still creates/destroys universes (unchanged)

### Container Image

A custom base image ships with:

- Ubuntu 24.04 base
- Claude Code CLI pre-installed
- Container-side Gate binary (Rust, using the official `agent-client-protocol` crate)
- Ready to communicate on container start—no setup delay

## Consequences

### What Becomes Easier

- **Multi-agent support.** Swapping Claude Code for Codex CLI or Gemini CLI is a container image change, not a protocol change. ACP abstracts the agent interface.
- **Session management.** ACP handles session create/load/resume natively. No custom `--resume` flag management.
- **Streaming output.** ACP's `session/update` events give real-time agent output through the Gate without custom log tailing.
- **Authentication.** ACP's `authenticate` flow handles API key setup inside the container.
- **Ecosystem alignment.** ACP is becoming the standard for agent communication. 34+ agents support it today.

### What Becomes Harder

- **Additional dependency.** ACP client and protocol add complexity inside the container image.
- **Protocol translation.** The Gate must translate between Spwn's protocol and ACP—two wire formats to maintain.
- **ACP evolution.** If ACP changes significantly, the container-side client must be updated.

## Alternatives Considered

| Alternative                       | Why Rejected                                                                                                                                                                                                               |
| --------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Adopt ACP end-to-end**          | Would make the Architect an ACP client directly. But ACP is designed for editor-to-agent communication, not orchestrator-to-container. The Gate protocol handles concerns ACP doesn't (mounts, element bridging, physics). |
| **Build custom agent protocol**   | Months of work to standardize what ACP already provides. We'd miss the 34+ agent ecosystem.                                                                                                                                |
| **Direct CLI spawning (current)** | Works for Claude Code but breaks when switching agents. No standard session/streaming/auth interface.                                                                                                                      |
| **ACP on the host only**          | Would require exposing ACP through the container boundary directly, complicating the security model. Gate-as-bridge keeps the boundary clean.                                                                              |
