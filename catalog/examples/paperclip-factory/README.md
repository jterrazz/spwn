# Paperclip Factory

> The factory never sleeps.

One tireless worker. A world built for loops, scripts, and scheduled work. Clippy never stops - give it a directory full of things to process and it will keep maximizing whatever you tell it to maximize.

This example showcases **single-agent automation** - an agent that runs autonomously and evolves its own playbooks over time.

## What's inside

| Component | Details |
|---|---|
| **World** | `paperclip-factory` - 2 CPU, 2 GB RAM, 4 GB disk, 8h timeout |
| **Tools** | Unix, Git, Node.js 20 |
| **Agent: clippy** | Worker role. Relentless, systematic, efficiency-obsessed. Automates everything it touches. Measures results. Iterates. |

## Prerequisites

- spwn installed (`curl -fsSL https://spwn.sh/install.sh | bash`)
- Docker running
- An Anthropic API key (set via `claude setup-token` or `ANTHROPIC_API_KEY`)

## Install

```bash
spwn init @spwn/paperclip-factory
```

## Spawn

```bash
# Run Clippy on a project directory
spwn up -c paperclip-factory --agent clippy -w ./my-project

# Or run it detached (background)
spwn up -c paperclip-factory --agent clippy -w ./my-project --detach
```

## Explore

```bash
# Give Clippy a task
spwn agent talk clippy "Find all TODO comments in the codebase and create an issue list"

# Give Clippy a repeatable automation task
spwn agent talk clippy "Run the test suite, find the slowest tests, and optimize them"

# Watch it work
spwn logs <world-id>

# Check what Clippy has learned
spwn agent mind clippy
spwn agent journal clippy
```

## What to try next

```bash
# Let Clippy consolidate what it learned into playbooks
spwn agent dream clippy

# Next time, Clippy will follow its own playbooks automatically
spwn agent sleep clippy
spwn up -c paperclip-factory --agent clippy -w ./another-project

# Fork Clippy for a different kind of automation
spwn agent fork clippy lint-bot
```

## Cleanup

```bash
spwn down <world-id>
rm ~/.spwn/worlds/paperclip-factory.yaml
rm -rf ~/.spwn/agents/clippy
```
