---
title: "spwn world send"
slug: "spwn-world-send"
---

## spwn world send

Send a message between agents in a world

### Synopsis

Send a message to an agent's inbox inside a running world.

```
spwn world send <world-id> [message] [flags]
```

### Options

```
      --from string   Sender agent name (required)
  -h, --help          help for send
      --to string     Recipient agent name (required)
      --type string   Message type: task, reply, question, announcement (default "task")
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn world](/docs/cli/spwn-world)	 - Spawn a world — an isolated reality for agents

