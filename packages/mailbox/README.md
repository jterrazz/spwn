# packages/mailbox

Filesystem-backed agent-to-agent messaging.

## Role

Agents inside a world drop messages for each other into per-agent inbox directories under `~/.spwn/`. No daemon, no queue — just JSON files on disk, read/written through a narrow CRUD API. Powers `spwn agent send`, `spwn agent inbox`, and `spwn agent watch`. Kept separate from the agent package so messaging can evolve (broker, transport, TTL) without disturbing mind/manifest concerns.

## Key types

- `Message` — one inbox entry: from/to, content, type, read flag, timestamps.
- `Send(inboxDir, from, to, content, msgType) → *Message` — append a message to the recipient's inbox.
- `Check(inboxDir, agentName)` / `CheckUnread` — list messages for an agent, optionally unread-only.
- `MarkRead` — flip the read flag.
- `ListAll(inboxDir)` — walk every agent's inbox (used by the web UI's global view).

## Related

- **Imported by** — `apps/cli`
- **Imports** — internal sub-packages (`inbox/`, `models/`)
