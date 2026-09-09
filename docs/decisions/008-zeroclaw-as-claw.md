# ADR-008: ZeroClaw as the Architect Daemon

**Status:** Accepted
**Date:** 2026-03-29

## Context

Spwn needs an always-on daemon (Architect) that:

1. Receives tasks from multiple messaging channels (Telegram, Slack, WhatsApp, Discord, CLI).
2. Creates and destroys universes.
3. Routes tasks to Governors.
4. Aggregates results and delivers them back through the originating channel.

This is a long-running, resource-sensitive process that must be reliable and lightweight.

## Decision

Use **ZeroClaw** as the implementation for the Architect daemon.

ZeroClaw is a <5MB Rust binary that supports 50+ messaging channels and multi-agent orchestration. It runs in its own Docker container with `/var/run/docker.sock` mounted, creating sibling world containers on the host's Docker daemon (Docker-out-of-Docker / DooD).

Key properties:

- **Tiny footprint**: <5MB binary, <5MB RAM at idle. Negligible resource overhead.
- **Rust**: memory-safe, no garbage collector pauses, ideal for a long-running daemon.
- **50+ channels**: Telegram, Slack, WhatsApp, Discord, and many more out of the box.
- **Multi-agent orchestration**: built-in support for task routing and agent coordination.

## Consequences

### Positive

- **Lightweight**: Architect adds negligible overhead to the host machine.
- **Reliable**: Rust's memory safety guarantees prevent crashes from memory bugs.
- **Channel breadth**: 50+ channels means users can interact with Claw from almost any platform.
- **Docker socket mounting natural fit**: ZeroClaw runs in a container and creates sibling containers via the mounted Docker socket, matching Spwn's DooD isolation model.

### Negative

- **Rust dependency**: the Architect daemon is Rust while the rest of Spwn core is Go. Two languages in the stack.
- **ZeroClaw coupling**: Spwn depends on an external project for the Architect daemon.

### Alternatives Considered

- **Custom Go daemon**: would unify the language stack but requires building messaging channel support from scratch.
- **Node.js bot framework**: mature messaging support but heavier runtime and worse resource profile for an always-on daemon.
