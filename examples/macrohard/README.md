# Macrohard

> Colony is alive.

Three agents, one world, one shared roster. A chief assigns work, two
workers deliver it, everyone talks through per-world inboxes.

## What's inside

- **World** `macrohard` — 4 CPU, 4 GB RAM, developer toolchain.
- **Agents**
  - `ballmer` — **chief**. Decomposes the backlog, hands out tasks,
    aggregates results. Shouts "DEVELOPERS!" internally.
  - `gates` — **worker**. Systems and backend focus.
  - `nadella` — **worker**. Frontend and integrations focus.

## Try it

```sh
spwn up -c macrohard --agents ballmer,gates,nadella
spwn agent talk ballmer "we're shipping a url shortener. break it down."
```

Ballmer decomposes the task, delegates to gates and nadella via their
per-world inboxes. Each worker reads its inbox on next invocation and
picks up where the chief left them.

## How the three talk

- Ballmer writes to `/agents/gates/worlds/<world-id>/inbox/…`
- Gates reads `/agents/gates/worlds/<world-id>/inbox/` on each turn
- Everybody sees `/world/roster.md` to know who else is in here

## Remove

```sh
rm ~/.spwn/worlds/macrohard.yaml
rm -rf ~/.spwn/agents/{ballmer,gates,nadella}
```
