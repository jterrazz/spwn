---
title: "spwn agent talk"
slug: "spwn-agent-talk"
---

## spwn agent talk

Talk to a running agent - interactive or one-shot

### Synopsis

Open a conversation with a named agent running inside a world.

If a message is provided, runs a one-shot query and prints the response.
If no message is provided, opens an interactive session inside the container.

```
spwn agent talk <agent-name> [message] [flags]
```

### Options

```
  -h, --help                   help for talk
      --output-format string   Output format: text (default) or stream-json
      --world string           World ID to target (disambiguates when the same agent exists in multiple worlds)
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent - a living identity that inhabits a world

