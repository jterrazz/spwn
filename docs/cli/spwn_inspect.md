---
title: "spwn inspect"
slug: "spwn-inspect"
---

## spwn inspect

Show per-agent composition: deps, skills, hooks

### Synopsis

Inspect a spwn project: one block per agent, showing the
resolved dependency tree (with transitive (*)-dedup and composition
badges), the skills contributed by tool deps, and the hooks bound
to the agent.

Mirrors the kubectl describe / cargo tree convention: key-value
header, section titles with counts, whitespace-separated sections.

Examples:
  spwn inspect            # every agent
  spwn inspect neo        # one agent
  spwn inspect --offline  # skip live world-status lookup

```
spwn inspect [agent] [flags]
```

### Options

```
  -h, --help      help for inspect
      --offline   Skip live world-status lookup (faster, no Docker calls)
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think

