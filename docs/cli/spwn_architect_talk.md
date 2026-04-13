---
title: "spwn architect talk"
slug: "spwn-architect-talk"
---

## spwn architect talk

Talk to the Architect — ask it to manage worlds and agents

### Synopsis

Send a message to the Architect (Claude Code running inside Docker).

If a message is provided, runs a one-shot query and prints the response.
If no message is provided, opens an interactive Claude session.

Examples:
  spwn architect talk "list all agents"
  spwn architect talk "create a new agent called neo"
  spwn architect talk                    # interactive mode

```
spwn architect talk [message] [flags]
```

### Options

```
  -h, --help                   help for talk
      --output-format string   Output format: text (default) or stream-json
```

### SEE ALSO

* [spwn architect](./spwn_architect.md)	 - Your always-on world builder

