---
title: "spwn agent talk"
slug: "spwn-agent-talk"
---

## spwn agent talk

Talk to a running agent — interactive or one-shot

### Synopsis

Open a conversation with a named agent running inside a world.

If a message is provided, runs a one-shot query and prints the response.
If no message is provided, opens an interactive Claude session inside the container.

```
spwn agent talk <agent-name> [message] [flags]
```

### Options

```
  -h, --help   help for talk
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn agent](/docs/cli/spwn-agent)	 - Spawn an agent — a living identity that inhabits a world

