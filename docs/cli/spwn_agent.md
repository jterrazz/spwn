---
title: "spwn agent"
slug: "spwn-agent"
---

## spwn agent

Spawn an agent - a living identity that inhabits a world

### Synopsis

Spawn an agent into an existing world.

An agent is backed by a Mind - a persistent directory holding its identity,
skills, playbooks, journal entries, and session state. The agent survives
after the world is destroyed. Knowledge lives at /world/knowledge/ inside
each world and is committed under spwn/worlds/<name>/knowledge/ — shared
across every agent in that world.

```
spwn agent [name] [flags]
```

### Examples

```
  spwn agent create neo              Create a blank agent
  spwn agent neo                     Open an interactive session with neo
  spwn agent -n neo -u w-abc123      Legacy: spawn named agent into world
  spwn agent --ephemeral "run tests"  Fire-and-forget ephemeral task
```

### Options

```
      --ephemeral string   Run as ephemeral agent - no Mind, no memory, just execute this task
  -h, --help               help for agent
      --import string      Import Mind from tar.gz before spawning
  -n, --name string        Agent name (default: default)
  -u, --world string       Target world ID
```

### SEE ALSO

* [spwn](./spwn.md)	 - spwn - create realities for things that can think
* [spwn agent add](./spwn_agent_add.md)	 - Add dependencies to an agent
* [spwn agent create](./spwn_agent_create.md)	 - Create a new agent (SOUL.md + 3-layer Mind)
* [spwn agent deploy](./spwn_agent_deploy.md)	 - Deploy an agent to a running world
* [spwn agent dream](./spwn_agent_dream.md)	 - Analyze experience, discover patterns, promote playbooks
* [spwn agent export](./spwn_agent_export.md)	 - [experimental] Export an agent as tar.gz
* [spwn agent fork](./spwn_agent_fork.md)	 - [experimental] Clone a Mind from one agent to another
* [spwn agent get](./spwn_agent_get.md)	 - Install a shared agent from the registry
* [spwn agent import](./spwn_agent_import.md)	 - [experimental] Import an agent from a tar.gz archive
* [spwn agent inbox](./spwn_agent_inbox.md)	 - [experimental] Show messages in an agent's inbox
* [spwn agent inspect](./spwn_agent_inspect.md)	 - Inspect an agent - composition, memory, world status, history
* [spwn agent logs](./spwn_agent_logs.md)	 - Show the event log for a specific agent
* [spwn agent ls](./spwn_agent_ls.md)	 - List all agents on this Host
* [spwn agent publish](./spwn_agent_publish.md)	 - Publish an agent to the registry (memory stripped)
* [spwn agent remove](./spwn_agent_remove.md)	 - Remove dependencies from an agent
* [spwn agent rm](./spwn_agent_rm.md)	 - Remove an agent and its Mind directory
* [spwn agent send](./spwn_agent_send.md)	 - [experimental] Send a message to an agent's inbox
* [spwn agent sleep](./spwn_agent_sleep.md)	 - Consolidate experience - archive stale files, prune old sessions
* [spwn agent start](./spwn_agent_start.md)	 - Start an agent as a background daemon [planned]
* [spwn agent stop](./spwn_agent_stop.md)	 - Stop an agent daemon [planned]
* [spwn agent talk](./spwn_agent_talk.md)	 - Talk to a running agent - interactive or one-shot
* [spwn agent watch](./spwn_agent_watch.md)	 - [experimental] Watch for new messages to an agent

