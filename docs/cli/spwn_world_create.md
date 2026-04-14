---
title: "spwn world create"
slug: "spwn-world-create"
---

## spwn world create

Declare a new world in spwn.yaml

### Synopsis

Append a worlds.<name> entry to the project's spwn.yaml.

This is a pure config write - no container is created. Once declared,
spawn the world with "spwn world start <name>" or "spwn world <name>".

The agents listed via --agent must already exist on disk under
spwn/agents/<name>/. Use "spwn agent create" first if they don't.

```
spwn world create <name> [flags]
```

### Examples

```
  spwn world create matrix --agent neo --agent trinity
  spwn world create alignment --agent clippy --workspace ./data:/workspace/data
```

### Options

```
  -a, --agent stringArray       Agent name (repeatable). Must already exist under spwn/agents/
  -h, --help                    help for create
  -w, --workspace stringArray   Workspace mount. Forms: "path", "host:/workspace/name"
```

### SEE ALSO

* [spwn world](./spwn_world.md)	 - Manage worlds - ephemeral runtime instances for agents

