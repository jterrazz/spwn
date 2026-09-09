# ADR-004: Go for Core Library

**Date:** 2026-02-28
**Status:** Accepted

## Context

Spwn needs a core implementation language for the business logic: lifecycle management, Mind management, physics generation, element bridging, and backend orchestration. The original design used TypeScript, then briefly Rust. With the pivot to Claude Code as the agent runtime (ADR-003), the primary workload is Docker orchestration + CLI — not performance-critical computation.

This is infrastructure glue. The right language for Docker orchestration is the one the Docker ecosystem is built in.

## Decision

Use Go for the core library and CLI. The architecture follows the `containerd` model:

```
┌────────────────────────┐  ┌──────────────────────────┐
│     spwn CLI            │  │       Thin SDKs           │
│  (Go, cobra — primary   │  │  (TS, Python — ~50 lines) │
│   interface)            │  │   Import domains directly  │
└───────────┬─────────────┘  └────────────┬─────────────┘
            │                             │
            └──────────┬──────────────────┘
                       ▼
┌──────────────────────────────────────────────────────┐
│               Domain Modules                          │
│     (Go modules — each with public API + internal/)   │
│  core/universe · core/agent · core/gate                │
└──────────────────────────────────────────────────────┘
```

### Why Go

| Factor                | Go                             | Rust                              | TypeScript              |
| --------------------- | ------------------------------ | --------------------------------- | ----------------------- |
| **Docker SDK**        | Official (docker/docker)       | bollard (third-party)             | dockerode (third-party) |
| **Ecosystem fit**     | Docker, containerd, K8s—all Go | Third-party wrappers              | Third-party wrappers    |
| **Compilation speed** | ~2s                            | ~30s+                             | N/A                     |
| **Single binary**     | Yes                            | Yes                               | No (needs Node.js)      |
| **Concurrency**       | Goroutines—simple, powerful    | tokio—powerful but ceremony-heavy | Event loop              |
| **Contributor pool**  | Large                          | Small                             | Large                   |
| **Distribution**      | `go install` or single binary  | `cargo install` or single binary  | `npm install`           |

### Key Go Dependencies

| Package         | Purpose                             |
| --------------- | ----------------------------------- |
| `docker/docker` | Docker Engine API client (official) |
| `spf13/cobra`   | CLI argument parsing                |
| `google/uuid`   | UUID generation                     |

## Consequences

### What Becomes Easier

- **Docker integration.** Official SDK, first-party documentation, same language as Docker itself.
- **Fast iteration.** 2-second builds vs 30+ seconds with Rust. Rapid prototyping.
- **Single binary distribution.** `spwn` is one binary—no runtime.
- **Contributors.** Go is widely known, lower barrier to entry.
- **Concurrency.** Goroutines are natural for container lifecycle management.

### What Becomes Harder

- **FFI for SDKs.** cgo is more complex than Rust's C ABI, adding friction for TS/Python bindings.
- **Type safety.** Go's type system is weaker than Rust's. No sum types, no ownership model.
- **Error handling.** Go's error returns are easy to accidentally ignore (though `errcheck` linter helps).

## Alternatives Considered

| Alternative    | Why Rejected                                                                                                                                                 |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Rust**       | Excellent language, but adds unnecessary complexity (async-trait, lifetimes, slow compilation) for what is Docker API orchestration. Third-party Docker SDK. |
| **TypeScript** | Requires Node.js runtime. Not suitable as infrastructure. Can't produce a single binary.                                                                     |
| **Python**     | Not suited for infrastructure. GIL, no single binary, slow.                                                                                                  |
