---
title: "spwn agent deploy"
slug: "spwn-agent-deploy"
---

## spwn agent deploy

Deploy an agent to a running world

### Synopsis

Adds an agent to an already-running world. The agent's mind is
mounted and a Claude Code session starts in the background.

The world must be running (idle or active). The agent must not already
be deployed in that world.

```
spwn agent deploy <agent-name> <world-id> [flags]
```

### Examples

```
  spwn agent deploy neo w-mars-47965
  spwn agent deploy morpheus w-mars-47965 --role chief
```

### Options

```
  -h, --help          help for deploy
      --role string   Agent role in the world organization (default "worker")
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent — a living identity that inhabits a world

