---
title: "spwn agent import"
slug: "spwn-agent-import"
---

## spwn agent import

[experimental] Import an agent from a tar.gz archive

### Synopsis

Import an agent's Mind from a previously exported tar.gz archive.

The agent name is derived from the archive filename (e.g., neo.tar.gz → neo).
The archive must contain at least a SOUL.md at the agent root.

⚠ Experimental: this command is in development and may change or break without notice.

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
      --as string   Rename the agent on import (instead of using the archive filename)
  -h, --help        help for import
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent - a living identity that inhabits a world

