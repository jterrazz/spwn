# The Matrix

> There is no spoon.

The simplest possible spwn world: one agent, one sandbox, no project.
Designed to be the first thing a new user spawns — talk to Neo, watch
it run commands, understand the model.

## What's inside

- **World** `matrix` — 2 CPU, 2 GB, full Unix toolchain, no project mount.
- **Agent** `neo` — a curious, low-ego agent that explains what it's
  doing as it does it. Great for first-time users who want a tour.

## Try it

```sh
spwn up -c matrix --agent neo
spwn agent talk neo "show me what you can see. explore the world."
```

Neo will walk you through `/world/`, `/agents/`, its own memory, and
what tools are available. It's essentially a self-documenting tour.

## After the tour

Once you're comfortable, destroy the matrix and spawn something real:

```sh
spwn down <world-id>
spwn example install paperclip-factory
```

## Remove

```sh
rm ~/.spwn/worlds/matrix.yaml
rm -rf ~/.spwn/agents/neo
```
