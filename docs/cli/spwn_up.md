---
title: "spwn up"
slug: "spwn-up"
---

## spwn up

Spawn a world — an isolated reality for agents

### Synopsis

Spawn a world — the Big Bang.

Creates an isolated Docker environment and brings an agent to life inside it.
Uses a named world config from ~/.spwn/worlds/ (default: default.yaml).

```
spwn up [flags]
```

### Examples

```
  spwn up -w .                    Spawn with current directory
  spwn up -c acme -w ~/project   Named config + workspace
  spwn up --leader morpheus       With a leader agent
```

### Options

```
  -a, --agent string            Agent name (default "default")
  -c, --config string           Named world config (default: default)
      --gate stringArray        Bridge tool from Host: "source:as:cap1,cap2"
  -h, --help                    help for up
  -i, --interactive             Attach to agent interactively
      --leader string           Leader agent for this world (gets the top role in the organization)
      --no-agent                Create the world without spawning an agent
      --organization string     Organization to use for role assignment (default "default")
      --runtime string          Agent runtime (default "claude-code")
  -w, --workspace stringArray   Host directory to mount. Repeatable. Forms: "path", "name=path", "name=path:ro". Omit for ephemeral.
  -u, --world string            Explicit path to a YAML config file
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn — create realities for things that can think

