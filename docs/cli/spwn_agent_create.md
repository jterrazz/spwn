---
title: "spwn agent create"
slug: "spwn-agent-create"
---

## spwn agent create

Create a new agent (SOUL.md + 2-layer Mind)

### Synopsis

Create a new agent with a SOUL.md at the agent root and the
two Mind layer directories (playbooks/journal). Skills aren't a Mind
layer — they're build-time dependencies resolved via the skill: scheme
or shipped by tools, injected into /world/skills/ at image time.
Knowledge is world-scoped, not agent-scoped — it lives at
/world/knowledge/ when a world opts in via spwn.yaml's
worlds.<name>.knowledge key, which resolves to a host path under the
project root. If no name is provided, a random name is picked from a
curated dictionary.

With --force, an existing agent's Mind is re-scaffolded: any missing
files are recreated and the command exits zero even if the agent
already exists.

```
spwn agent create [name] [flags]
```

### Options

```
  -f, --force         Re-scaffold any missing Mind files without complaining if the agent already exists
  -h, --help          help for create
      --team string   Assign agent to a team (slug)
```

### SEE ALSO

* [spwn agent](./spwn_agent.md)	 - Spawn an agent - a living identity that inhabits a world

