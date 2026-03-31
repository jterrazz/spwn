---
title: "spwn agent send"
slug: "spwn-agent-send"
---

## spwn agent send

Send a message to an agent's inbox

### Synopsis

Send an async message to a running agent. The agent must be in an active world.

```
spwn agent send <agent-name> [message] [flags]
```

### Options

```
      --from string   Sender agent name (required)
  -h, --help          help for send
      --type string   Message type: task, reply, question, announcement (default "task")
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn agent](/docs/cli/spwn-agent)	 - Spawn an agent — a living identity that inhabits a world

