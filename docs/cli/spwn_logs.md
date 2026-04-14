---
title: "spwn logs"
slug: "spwn-logs"
---

## spwn logs

Show the system event log across worlds and agents

### Synopsis

Show the spwn event log - spawned worlds, created agents, dream cycles,
snapshots, messages, and every other discrete thing that happened.

Scope it with --world or --agent, or use the per-entity shortcuts:
  spwn world logs <id>        events for one world
  spwn agent logs <name>      events for one agent
  spwn architect logs         architect daemon events

```
spwn logs [flags]
```

### Options

```
  -a, --agent string   Filter by agent name
  -h, --help           help for logs
  -n, --limit int      Number of events to show (default 20)
  -t, --type string    Filter by event type (e.g. agent.dreamed)
  -w, --world string   Filter by world ID
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think

