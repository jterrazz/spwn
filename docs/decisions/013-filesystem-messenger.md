# ADR-013: Filesystem-Based Agent Messaging

**Status:** Accepted
**Date:** 2026-03-31

The record was written at decision time; only these two lines are
reconstructed, 2026-09-05, from the commit that carries it (`b824a17`,
"docs: ADR-012 filesystem-based messenger", 2026-03-31) — the same day the
message timestamps below were taken from.

## Context

Agents living in the same world need to communicate — governors need to delegate tasks, citizens need to report progress, and peers need to collaborate. We needed a messaging system that works inside Docker containers without external infrastructure.

## Decision

We implemented a **filesystem-based inbox system** at `/world/inbox/`. Each agent has their own inbox directory (`/world/inbox/{name}/`). Messages are JSON files written directly to the filesystem.

### How it works

- **Send**: Write a JSON file to `/world/inbox/{recipient}/msg-{id}.json`
- **Check**: Read files from `/world/inbox/{your-name}/`
- **Watch**: A polling daemon checks for new unread messages every 5 seconds and wakes agents via `spwn agent talk`

### Message format

```json
{
    "id": "msg-morpheus-20260331-083001-042",
    "from": "morpheus",
    "to": "neo",
    "timestamp": "2026-03-31T08:30:01Z",
    "type": "task",
    "content": "Implement the webhook handler",
    "status": "unread"
}
```

## Alternatives Considered

1. **Message broker (Redis/NATS)** — adds infrastructure dependency, heavier
2. **Unix domain sockets** — complex, harder to debug, not persistent
3. **HTTP API between agents** — needs a server per agent, networking overhead
4. **Shared database (SQLite)** — single-writer lock contention

## Consequences

### Positive

- Zero infrastructure — just files
- Human-readable — `cat /world/inbox/neo/*.json`
- Naturally persistent — survives container restart
- Agents use normal tools (Read, Write) — no special SDK
- Searchable with standard Unix tools (`grep`, `find`)

### Negative

- Polling-based (not real-time) — 5-second latency
- No delivery guarantees (file write can fail silently)
- No message ordering guarantees beyond timestamp
- Scales poorly to thousands of messages (filesystem limits)

## Future

- Upgrade to inotify/fswatch for real-time notification
- Add message acknowledgment (delivered → read → processed)
- Consider SQLite for high-volume worlds
