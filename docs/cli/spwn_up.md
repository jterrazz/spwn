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
  spwn up --agent neo -w .                  Single agent in current dir
  spwn up --agent morpheus --agent neo -w .  Multi-agent (morpheus is chief)
  spwn up -c acme --agent neo -w ~/project   Named config + workspace
```

### Options

```
  -a, --agent stringArray       Agent name (repeatable; first agent becomes chief in multi-agent worlds)
  -c, --config string           Named world config (default: default)
      --gate stringArray        Bridge tool from Host: "source:as:cap1,cap2"
  -h, --help                    help for up
  -i, --interactive             Attach to agent interactively
      --no-agent                Create the world without spawning an agent
      --runtime string          Agent runtime (default "claude-code")
  -w, --workspace stringArray   Host directory to mount. Repeatable. Forms: "path", "name=path", "name=path:ro". Omit for ephemeral.
  -u, --world string            Explicit path to a YAML config file
```

### Options inherited from parent commands

```
      --json   Output as JSON
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn — create realities for things that can think

