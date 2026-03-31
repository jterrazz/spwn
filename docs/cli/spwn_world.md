---
title: "spwn world"
slug: "spwn-world"
---

## spwn world

Spawn a world — an isolated reality for agents

### Synopsis

Spawn a world — the Big Bang.

Creates an isolated Docker environment and brings an agent to life inside it.
Uses a named world config from ~/.spwn/worlds/ (default: default.yaml).

```
spwn world [flags]
```

### Examples

```
  spwn world -w .                    Spawn with current directory
  spwn world -c acme -w ~/project   Named config + workspace
  spwn world --governor morpheus     With a governor agent
  spwn world --no-agent              Empty world (no agent)
```

### Options

```
  -a, --agent string       Agent name (default "default")
  -c, --config string      Named world config (default: default)
      --gate stringArray   Bridge element from Host: "source:as:cap1,cap2"
      --governor string    Governor agent for this world
  -h, --help               help for world
  -i, --interactive        Attach to agent interactively
      --no-agent           Create the world without spawning an agent
      --runtime string     Agent runtime (claude-code, pi, codex, opencode, gemini, aider) (default "claude-code")
  -w, --workspace string   Host directory to mount at /workspace
  -u, --world string       Explicit path to a YAML config file
```

### Options inherited from parent commands

```
      --json      Output as JSON
  -q, --quiet     Suppress non-essential output
  -v, --verbose   Show debug information
```

### SEE ALSO

* [spwn](/docs/cli/spwn)	 - spwn — create realities for things that can think
* `spwn down <id>` — Destroy a world (top-level command)
* `spwn inspect <id>` — Show world details, physics, and agent status (top-level command)
* `spwn ls` — List all active worlds (top-level command)
* `spwn logs <id>` — Stream agent output from a running world (top-level command)
* `spwn attach <id>` — Open interactive session into a running world (top-level command)

