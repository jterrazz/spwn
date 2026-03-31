---
title: "spwn agent"
slug: "spwn-agent"
---

## spwn agent

Spawn an agent — a living identity that inhabits a world

### Synopsis

Spawn an agent into an existing world.

An agent is backed by a Mind — a persistent directory of personas, skills,
knowledge, playbooks, journal entries, and session state. The agent survives
after the world is destroyed.

```
spwn agent [flags]
```

### Examples

```
  spwn agent -n neo -u w-abc123      Spawn named agent into world
  spwn agent --npc "run tests"       Fire-and-forget NPC task
  spwn agent --import backup.tar.gz  Import a Mind archive first
```

### Options

```
  -h, --help            help for agent
      --import string   Import Mind from tar.gz before spawning
  -n, --name string     Agent name (default: default)
      --npc string      Run as NPC — no Mind, no memory, just execute this task
  -u, --world string    Target world ID
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn](/docs/cli/spwn)	 - spwn — create realities for things that can think
* [spwn agent delete](/docs/cli/spwn-agent-delete)	 - Remove an agent and its Mind directory
* [spwn agent export](/docs/cli/spwn-agent-export)	 - Export an agent as tar.gz
* [spwn agent fork](/docs/cli/spwn-agent-fork)	 - Clone a Mind from one agent to another
* [spwn agent init](/docs/cli/spwn-agent-init)	 - Create a new agent with a 6-layer Mind
* [spwn agent inspect](/docs/cli/spwn-agent-inspect)	 - Show agent details, Mind layers, world status, and history
* [spwn agent journal](/docs/cli/spwn-agent-journal)	 - View an agent's journal history
* [spwn agent list](/docs/cli/spwn-agent-list)	 - List all agents on this Host
* [spwn agent reflect](/docs/cli/spwn-agent-reflect)	 - Analyze journal and promote patterns to playbooks
* [spwn agent sessions](/docs/cli/spwn-agent-sessions)	 - View an agent's session history
* [spwn agent sleep](/docs/cli/spwn-agent-sleep)	 - Consolidate experience — archive stale files, prune old sessions
* [spwn agent talk](/docs/cli/spwn-agent-talk)	 - Talk to a running agent — interactive or one-shot

