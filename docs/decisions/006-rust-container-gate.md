# ADR-006: Rust for the Container-Side Gate

**Date:** 2026-03-01
**Status:** Accepted

## Context

The container-side Gate is a binary that runs inside the universe container. It speaks ACP (JSON-RPC over stdio) to the agent CLI, handles crash recovery, and communicates with the host-side Gate over TCP. The rest of Spwn is written in Go.

The question: what language should the container-side Gate be written in?

## Options Considered

| Language       | ACP SDK                                | Binary Size     | Container Dependency | Trade-off                                                                                                          |
| -------------- | -------------------------------------- | --------------- | -------------------- | ------------------------------------------------------------------------------------------------------------------ |
| **Go**         | No official SDK                        | ~5-10 MB static | None                 | Same language as core, but must hand-roll ACP JSON-RPC client (~200-300 lines) and track protocol changes manually |
| **Rust**       | Official `agent-client-protocol` crate | ~2-5 MB static  | None                 | Second language/toolchain, but full ACP implementation for free                                                    |
| **TypeScript** | Official `@agentclientprotocol/sdk`    | ~50+ MB         | Node.js runtime      | Adds heavy runtime dependency to every container                                                                   |
| **Python**     | Official `python-sdk`                  | ~30+ MB         | Python runtime       | Adds heavy runtime dependency to every container                                                                   |

## Decision

Use **Rust** with the official [`agent-client-protocol`](https://crates.io/crates/agent-client-protocol) crate.

The container-side Gate lives at `platform/gate-runtime/` as a standalone Rust crate, built as a static binary (`spwn-gate`), and copied into the container image at build time.

### Why Rust

1. **Official ACP SDK.** The `agent-client-protocol` crate gives us the full ACP implementation—session management, streaming, authentication, JSON-RPC framing—without reimplementing it. Protocol changes upstream are absorbed via `cargo update`.

2. **ACP is still evolving.** New methods, session semantics, and streaming changes arrive regularly. With Go, we'd track and reimplement each change manually. With Rust, we update the crate.

3. **Small static binary.** ~2-5 MB, zero runtime dependencies. Ideal for containers where image size matters.

4. **Clean boundary.** The container-side Gate communicates with the host-side Gate over TCP. It doesn't share code, types, or state with the Go codebase. It's a self-contained binary with a well-defined interface—more like a dependency than a module.

5. **Minimal surface area.** The container-side Gate does exactly three things: speak ACP to the agent CLI, supervise the agent process, and listen for commands from the host-side Gate. It's small enough that the second-language cost is negligible.

### Why Not Go

Go would keep the project as a single language. But the container-side Gate is the one place where ACP protocol fidelity matters most—it's the component that spawns agents, manages sessions, handles streaming, and recovers from crashes. Reimplementing ACP's JSON-RPC protocol in Go (~200-300 lines) is achievable, but tracking upstream protocol changes manually introduces ongoing maintenance burden and risk of incompatibility.

## Consequences

### What Becomes Easier

- **Protocol fidelity.** The official SDK is always correct and up-to-date with ACP spec changes.
- **Maintenance.** `cargo update` absorbs ACP protocol evolution. No manual JSON-RPC tracking.
- **Container image.** Smaller binary than Go equivalent. No runtime dependencies.

### What Becomes Harder

- **Two build toolchains.** `go build` for the core + `cargo build` for the container-side Gate. CI must handle both.
- **Developer onboarding.** Contributors need both Go and Rust toolchains installed.
- **Code sharing.** Types and constants can't be shared between Go core and Rust gate. The TCP wire format is the contract.

### Mitigation

The container-side Gate is intentionally small and isolated. The Rust surface area is one crate with one binary. The boundary is TCP with a simple HTTP wire format. Most development happens in the Go core—the Rust gate is built once and updated primarily when ACP evolves.
