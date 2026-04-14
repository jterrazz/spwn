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
      --from string   Sender name (default: user) (default "user")
  -h, --help          help for send
      --type string   Message type: task, reply, question, announcement (default "task")
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent - a living identity that inhabits a world

