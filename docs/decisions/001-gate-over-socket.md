# ADR-001: Gate Abstraction Over Direct Transport

**Date:** 2025-02-05
**Status:** Accepted

## Context

The SDK needs to communicate with processes running inside Universes. Docker provides `docker exec` and the Docker API for this, but if the SDK communicates directly using Docker-specific transports, it becomes tightly coupled to the backend. We need to decide whether to expose raw transports or abstract them behind a unified protocol.

## Decision

We introduce the **Gate**—an abstraction layer that sits between the SDK and the transport. The SDK always talks to a Gate. The Gate translates to the appropriate transport for the active backend.

```text
┌──────────┐       ┌──────────┐       ┌──────────────┐
│   SDK    │──────►│   Gate   │──────►│   Backend    │
│          │       │          │       │  (transport) │
│ exec()   │       │ encode   │       │  docker exec │
│ read()   │       │ route    │       │              │
│ write()  │       │ decode   │       │              │
└──────────┘       └──────────┘       └──────────────┘
```

The Gate defines a small set of **message types**:

| Message Type | Direction       | Purpose                                       |
| ------------ | --------------- | --------------------------------------------- |
| `exec`       | Host → Universe | Run a command, return stdout/stderr/exit code |
| `file_read`  | Host → Universe | Read a file from the Universe filesystem      |
| `file_write` | Host → Universe | Write a file to the Universe filesystem       |
| `signal`     | Host → Universe | Send a signal to a running process            |
| `heartbeat`  | Bidirectional   | Keep-alive, health check                      |
| `stream`     | Universe → Host | Real-time output from a running command       |

## Rationale

The Gate is a thin layer—it adds minimal overhead but provides critical decoupling. This is the same pattern as database drivers (SQL is the gate, the driver handles transport) or HTTP (the protocol is the gate, TCP/TLS is the transport). The abstraction keeps the door open for future backends without changing agent code.

## Consequences

### Positive

- **Backend portability.** Agent code doesn't depend on Docker directly.
- **Testability.** Mock the Gate for unit testing agents without spinning up containers.
- **Extensibility.** New backends only need a transport adapter, not SDK changes.
- **Structured communication.** Message types enforce a contract—no more parsing raw shell output for control flow.

### Negative

- **Abstraction cost.** The Gate adds a translation step. For Docker exec, this is nearly zero.
- **Least common denominator.** Features unique to one transport (e.g., Docker API's container stats) aren't exposed through the Gate unless explicitly added.

### Mitigations

- Keep the Gate protocol minimal—only the operations agents actually need.
- Allow escape hatches for backend-specific features when needed.

## Alternatives Considered

| Alternative                 | Why Rejected                                                  |
| --------------------------- | ------------------------------------------------------------- |
| **Direct transport access** | SDK code becomes backend-specific, breaks on migration        |
| **gRPC**                    | Heavy dependency, overkill for the message set                |
| **REST over TCP**           | Requires networking setup in every Universe                   |
| **Custom binary protocol**  | Harder to debug, no ecosystem tooling, premature optimization |
