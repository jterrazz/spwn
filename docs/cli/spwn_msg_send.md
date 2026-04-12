---
title: "spwn msg send"
slug: "spwn-msg-send"
---

## spwn msg send

Send a message to an agent's inbox

### Synopsis

Send an async message to a running agent. The agent must be in an active world.

```
spwn msg send <agent-name> [message] [flags]
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

* [spwn msg](./spwn_msg.md)	 - Agent messaging — send, inbox, watch

