---
title: "spwn up"
slug: "spwn-up"
---

## spwn up

Spawn a world - an isolated reality for agents

### Synopsis

Spawn a world - the Big Bang.

Creates an isolated Docker environment. Pass --agent (repeatable) to bring
agents to life inside it. Without any --agent flag, the world spawns empty.

```
spwn up [flags]
```

### Examples

```
  spwn world up --agent neo -w .                  Single agent in current dir
  spwn world up --agent morpheus --agent neo -w .  Multi-agent (morpheus is chief)
  spwn world up --name "Big Refactor" --agent neo  Ephemeral (no host mount)
  spwn world up                                    Empty world (no agent)
```

### Options

```
  -a, --agent stringArray       Agent name (repeatable; first agent becomes chief in multi-agent worlds)
      --build                   Run spwn build first, then spawn from the artifact
  -c, --config string           Named world config (default: default)
  -h, --help                    help for up
  -i, --interactive             Drop into the agent's session after spawn
  -n, --name string             Display name for the world
  -w, --workspace stringArray   Host directory to mount. Repeatable. Forms: "path", "name=path", "name=path:ro". Omit for ephemeral.
  -u, --world string            Explicit path to a YAML config file
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think

