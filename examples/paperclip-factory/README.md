# Paperclip Factory

> The factory never sleeps.

A single-agent workshop. One world, one worker, no ceremony. Point it at a
directory, describe what needs to happen, and let it loop.

## What's inside

- **World** `paperclip-factory` — 2 CPU, 2 GB RAM, Unix + Git + Node tools.
- **Agent** `clippy` — a relentless worker agent tuned for automation
  chores: script generation, batch processing, scheduled tasks.

## Try it

After installing this example from the observatory gallery (or
`spwn example install paperclip-factory`), run:

```sh
spwn up -c paperclip-factory --agent clippy
```

Then talk to it:

```sh
spwn agent talk clippy "find every PNG in ~/Downloads and resize to 512px"
```

## Customize

The world config is at `~/.spwn/worlds/paperclip-factory.yaml`. Add more
tools, bump the physics constants, or point the agent at a real workspace:

```sh
spwn up -c paperclip-factory --agent clippy -w ~/my-project
```

## Remove

```sh
rm ~/.spwn/worlds/paperclip-factory.yaml
rm -rf ~/.spwn/agents/clippy
```
