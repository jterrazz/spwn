---
title: "spwn world up"
slug: "spwn-world-up"
---

## spwn world up

Spawn a world - an isolated reality for agents

### Synopsis

Spawn a world - the Big Bang.

Inside a spwn project:
  spwn up             brings up every world declared in spwn.yaml
  spwn up <name>      brings up a specific world from spwn.yaml

Outside a project, the legacy global-mode flags still work and spawn
a one-off world from ~/.spwn/worlds/<config>.yaml.

```
spwn world up [name] [flags]
```

### Examples

```
  spwn up                                          Bring up every world in spwn.yaml
  spwn up neo                                      Start the "neo" world
  spwn world up --agent neo -w .                  Single agent in current dir
  spwn world up --agent morpheus --agent neo -w .  Multi-agent (morpheus is chief)
```

### Options

```
  -a, --agent stringArray       Agent name (repeatable; first agent becomes chief in multi-agent worlds)
  -c, --config string           Named world config (default: default)
      --force-rebuild           Ignore the image cache and rebuild the world image from scratch
  -h, --help                    help for up
  -i, --interactive             Drop into the agent's session after spawn
  -n, --name string             Display name for the world
  -w, --workspace stringArray   Host directory to mount. Repeatable. Forms: "path", "name=path", "name=path:ro". Omit for ephemeral.
  -u, --world string            Explicit path to a YAML config file
```

### SEE ALSO

* [spwn world](./spwn_world.md)	 - Manage worlds - ephemeral runtime instances for agents

