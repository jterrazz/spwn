---
title: "spwn world start"
slug: "spwn-world-start"
---

## spwn world start

Start a world (alias for `spwn up`)

```
spwn world start [name] [flags]
```

### Options

```
  -a, --agent stringArray       Agent name (repeatable; first agent becomes chief in multi-agent worlds)
      --backend string          Override the runtime backend for this spawn (e.g. claude-code, codex). Skips the auth-state auto-resolver.
  -c, --config string           Named world config (default: default)
      --force-rebuild           Ignore the image cache and rebuild the world image from scratch
  -h, --help                    help for start
  -i, --interactive             Drop into the agent's session after spawn
  -n, --name string             Display name for the world
  -w, --workspace stringArray   Host directory to mount. Repeatable. Forms: "path", "name=path", "name=path:ro". Omit for ephemeral.
  -u, --world string            Explicit path to a YAML config file
```

### SEE ALSO

* [spwn world](./spwn_world.md)	 - Manage worlds - ephemeral runtime instances for agents

