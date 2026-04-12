---
title: "spwn agent import"
slug: "spwn-agent-import"
---

## spwn agent import

Import an agent from a tar.gz archive

### Synopsis

Import an agent's Mind from a previously exported tar.gz archive.

The agent name is derived from the archive filename (e.g., neo.tar.gz → neo).
The archive must contain at least an identity/ layer.

```
spwn agent import <path-to-tar.gz> [flags]
```

### Examples

```
  spwn agent import neo.tar.gz
  spwn agent import /path/to/backup.tar.gz
```

### Options

```
  -h, --help   help for import
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent — a living identity that inhabits a world

