# packages/activity

System-wide event log for spwn.

## Role

Every meaningful lifecycle event (world spawned, agent joined, session ended, …) is appended as a JSONL record to `~/.spwn/activity.jsonl`. The log is the single source of truth for "what happened on this machine, and when" — the CLI's `spwn logs` view and the web UI feed off it. Writes are best-effort: if the filesystem is unreachable the caller is never blocked, activity is just dropped.

## Key types

- `Event` — one record. Carries a `Type`, actor, target, free-form phrase, and timestamp.
- `Type` — dotted namespace (`world.spawned`, `agent.joined`, …). Stable string, consumed by filters.
- `Log(e Event)` — append a single event. Fills in ID + Timestamp if empty.
- `Phrase*` functions — pre-baked human-readable phrases, one per `Type`. The emitting code is responsible for supplying its own phrase at write time (centralised phrasing keeps the log readable).

## Related

- **Imported by** — `apps/api`, `apps/cli`, `packages/agent`, `packages/architect`
- **Imports** — `packages/platform` (for `~/.spwn/` paths)
