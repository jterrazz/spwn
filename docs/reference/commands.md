# Commands

Every command of the `spwn` binary, as `spwn <command> --help` prints it. The page is projected from the Cobra command tree by `make docs` and is never edited by hand; the grammar the commands follow is [the CLI](cli.md).

## spwn

spwn - create realities for things that can think.

```text
spwn creates isolated Docker environments for AI agents.
Each world has its own rules (network, filesystem, tools) and a
Mind (persistent agent identity).
```

- [spwn agent](#spwn-agent) — Spawn an agent - a living identity that inhabits a world.
- [spwn architect](#spwn-architect) — Your always-on world builder.
- [spwn auth](#spwn-auth) — Manage credentials — dashboard, login, use, logout, disable.
- [spwn automation](#spwn-automation) — Trigger-driven agent wakeups (cron + filesystem).
- [spwn build](#spwn-build) — Transpile the project and compile it into a Docker image.
- [spwn check](#spwn-check) — Validate the project tree against spwn.yaml.
- [spwn cookie-sync](#spwn-cookie-sync) — Browser extension that auto-syncs session cookies to the gate (status + providers).
- [spwn down](#spwn-down) — Destroy a world.
- [spwn gate](#spwn-gate) — Manage the host-side credential broker (start/stop/status/logs/restart).
- [spwn init](#spwn-init) — Scaffold a spwn project in the current directory.
- [spwn inspect](#spwn-inspect) — Show per-agent composition: deps, skills, hooks.
- [spwn install](#spwn-install) — Install a dependency into the project.
- [spwn logs](#spwn-logs) — Show the system event log across worlds and agents.
- [spwn ls](#spwn-ls) — Agent-centric status (running / stopped / orphan).
- [spwn skill](#spwn-skill) — Author bare-markdown skill files.
- [spwn status](#spwn-status) — Show the full status of your spwn environment.
- [spwn uninstall](#spwn-uninstall) — Remove a dependency from the project.
- [spwn up](#spwn-up) — Spawn a world - an isolated reality for agents.
- [spwn upgrade](#spwn-upgrade) — Upgrade spwn to the latest version.
- [spwn world](#spwn-world) — Manage worlds - ephemeral runtime instances for agents.

## spwn agent

Spawn an agent - a living identity that inhabits a world.

```text
Spawn an agent into an existing world.
```

```text
An agent is backed by a Mind - a persistent directory holding its SOUL.md
(purpose, voice, principles), playbooks/, and journal/. The agent survives
after the world is destroyed. Skills are build-time dependencies injected
into /world/skills/; knowledge lives at /world/knowledge/ inside each
world when the manifest opts in via worlds.<name>.knowledge — the path
resolves relative to the project root and is shared across every agent
in that world.
```

```text
spwn agent [name] [flags]
```

```text
      --backend string     Override the runtime backend for this session (e.g. claude-code, codex).
      --ephemeral string   Run as ephemeral agent - no Mind, no memory, just execute this task
      --import string      Import Mind from tar.gz before spawning
  -n, --name string        Agent name (default: default)
  -u, --world string       Target world ID
```

- [spwn agent create](#spwn-agent-create) — Create a new agent (SOUL.md + 2-layer Mind).
- [spwn agent deploy](#spwn-agent-deploy) — Deploy an agent to a running world.
- [spwn agent dream](#spwn-agent-dream) — Analyze experience, discover patterns, promote playbooks.
- [spwn agent export](#spwn-agent-export) — [experimental] Export an agent as tar.gz.
- [spwn agent fork](#spwn-agent-fork) — [experimental] Clone a Mind from one agent to another.
- [spwn agent import](#spwn-agent-import) — [experimental] Import an agent from a tar.gz archive.
- [spwn agent inspect](#spwn-agent-inspect) — Inspect an agent - composition, memory, world status, history.
- [spwn agent logs](#spwn-agent-logs) — Show the event log for a specific agent.
- [spwn agent ls](#spwn-agent-ls) — List all agents on this Host.
- [spwn agent rm](#spwn-agent-rm) — Remove an agent and its Mind directory.
- [spwn agent sleep](#spwn-agent-sleep) — Consolidate experience - archive stale files, prune old sessions.
- [spwn agent talk](#spwn-agent-talk) — Talk to a running agent - interactive or one-shot.

## spwn agent create

Create a new agent (SOUL.md + 2-layer Mind).

```text
Create a new agent with a SOUL.md at the agent root and the
two Mind layer directories (playbooks/journal). Skills aren't a Mind
layer — they're build-time dependencies resolved via the skill/ form
or shipped by tools, injected into /world/skills/ at image time.
Knowledge is world-scoped, not agent-scoped — it lives at
/world/knowledge/ when a world opts in via spwn.yaml's
worlds.<name>.knowledge key, which resolves to a host path under the
project root. If no name is provided, a random name is picked from a
curated dictionary.
```

```text
With --force, an existing agent's Mind is re-scaffolded: any missing
files are recreated and the command exits zero even if the agent
already exists.
```

```text
spwn agent create [name] [flags]
```

```text
  -f, --force         Re-scaffold any missing Mind files without complaining if the agent already exists
      --team string   Assign agent to a team (slug)
```

## spwn agent deploy

Deploy an agent to a running world.

```text
Adds an agent to an already-running world. The agent's mind is
mounted and a Claude Code session starts in the background.
```

```text
The world must be running (idle or active). The agent must not already
be deployed in that world.
```

```text
spwn agent deploy <agent-name> <world-id> [flags]
```

```text
      --role string   Agent role in the world organization (default "worker")
```

## spwn agent dream

Analyze experience, discover patterns, promote playbooks.

```text
spwn agent dream <agent-name>
```

## spwn agent export

[experimental] Export an agent as tar.gz.

```text
[experimental] Export an agent as tar.gz
```

```text
⚠ Experimental: this command is in development and may change or break without notice.
```

```text
spwn agent export <agent-name> [flags]
```

```text
      --exclude strings   Exclude layers (e.g., journal, sessions)
      --output string     Output directory (default ".")
```

## spwn agent fork

[experimental] Clone a Mind from one agent to another.

```text
[experimental] Clone a Mind from one agent to another
```

```text
⚠ Experimental: this command is in development and may change or break without notice.
```

```text
spwn agent fork <source> <target>
```

## spwn agent import

[experimental] Import an agent from a tar.gz archive.

```text
Import an agent's Mind from a previously exported tar.gz archive.
```

```text
The agent name is derived from the archive filename (e.g., neo.tar.gz → neo).
The archive must contain at least a SOUL.md at the agent root.
```

```text
⚠ Experimental: this command is in development and may change or break without notice.
```

```text
spwn agent import <path-to-tar.gz> [flags]
```

```text
      --as string   Rename the agent on import (instead of using the archive filename)
```

## spwn agent inspect

Inspect an agent - composition, memory, world status, history.

```text
spwn agent inspect <agent-name>
```

## spwn agent logs

Show the event log for a specific agent.

```text
spwn agent logs <agent-name> [flags]
```

```text
  -n, --limit int   Number of events to show (default 50)
```

## spwn agent ls

List all agents on this Host.

```text
spwn agent ls [flags]
```

```text
      --json           Emit results as structured JSON on stdout
      --world string   Filter agents by world ID
```

## spwn agent rm

Remove an agent and its Mind directory.

```text
spwn agent rm <agent-name>
```

## spwn agent sleep

Consolidate experience - archive stale files, prune old sessions.

```text
spwn agent sleep <agent-name>
```

## spwn agent talk

Talk to a running agent - interactive or one-shot.

```text
Open a conversation with a named agent running inside a world.
```

```text
If a message is provided, runs a one-shot query and prints the response.
If no message is provided, opens an interactive session inside the container.
```

```text
spwn agent talk <agent-name> [message] [flags]
```

```text
      --output-format string   Output format: text (default) or stream-json
      --world string           World ID to target (disambiguates when the same agent exists in multiple worlds)
```

## spwn architect

Your always-on world builder.

```text
The Architect is your always-on world builder. It manages worlds and orchestrates artificial life.
```

- [spwn architect logs](#spwn-architect-logs) — Show the Architect daemon's event log.
- [spwn architect start](#spwn-architect-start) — Start the Architect daemon.
- [spwn architect status](#spwn-architect-status) — Show Architect status - worlds, agents.
- [spwn architect stop](#spwn-architect-stop) — Stop the Architect daemon.
- [spwn architect talk](#spwn-architect-talk) — Talk to the Architect - ask it to manage worlds and agents.

## spwn architect logs

Show the Architect daemon's event log.

```text
spwn architect logs [flags]
```

```text
  -n, --limit int   Number of events to show (default 50)
```

## spwn architect start

Start the Architect daemon.

```text
Start the Architect daemon in a Docker container.
```

```text
The Architect runs the spwn binary inside a long-lived container with the
host's Docker socket mounted (DooD - Docker-outside-of-Docker), allowing it
to create and manage world containers as siblings.
```

```text
The container mounts:
  /var/run/docker.sock    Docker daemon access (sibling containers, not nested)
  ~/.spwn/                Shared configuration and state
```

```text
The Architect's identity is defined in /world/ARCHITECT.md inside the container,
which describes its capabilities and role as the always-on world builder.
```

```text
spwn architect start
```

## spwn architect status

Show Architect status - worlds, agents.

```text
Query Docker to show whether the Architect container is running, its uptime, and active worlds.
```

```text
spwn architect status
```

## spwn architect stop

Stop the Architect daemon.

```text
Stop the Architect daemon and remove the container.
```

```text
spwn architect stop
```

## spwn architect talk

Talk to the Architect - ask it to manage worlds and agents.

```text
Send a message to the Architect (Claude Code running inside Docker).
```

```text
If a message is provided, runs a one-shot query and prints the response.
If no message is provided, opens an interactive Claude session.
```

```text
Examples:
  spwn architect talk "list all agents"
  spwn architect talk "create a new agent called neo"
  spwn architect talk                    # interactive mode
```

```text
spwn architect talk [message] [flags]
```

```text
      --output-format string   Output format: text (default) or stream-json
```

## spwn auth

Manage credentials — dashboard, login, use, logout, disable.

```text
spwn auth
```

- [spwn auth default](#spwn-auth-default) — Pick which provider spwn prefers when multiple are authenticated.
- [spwn auth disable](#spwn-auth-disable) — Tell spwn not to use this provider, even if creds exist.
- [spwn auth enable](#spwn-auth-enable) — Reverse a previous `disable`.
- [spwn auth login](#spwn-auth-login) — Set up credentials for a provider.
- [spwn auth logout](#spwn-auth-logout) — Clear stored credentials for a provider.
- [spwn auth use](#spwn-auth-use) — Pick which credential method spwn should prefer.

## spwn auth default

Pick which provider spwn prefers when multiple are authenticated.

```text
Set a soft preference for which provider's runtime spwn picks
when you're logged into more than one and no runtime is pinned at the
project or agent level.
```

```text
This is the durable answer to the "multiple providers authenticated
and no runtime pinned" error — set it once and spwn will quietly
resolve ambiguity in that provider's favour. Does NOT disable the
other provider or override agent.yaml / spwn.yaml pins.
```

```text
Example:
  spwn auth default anthropic        # prefer claude-code
  spwn auth default --clear          # revert to auto-resolve
```

```text
spwn auth default [provider] [flags]
```

```text
      --clear   Remove the default preference (revert to auto-resolve)
```

## spwn auth disable

Tell spwn not to use this provider, even if creds exist.

```text
Opt a provider out without touching credentials. Useful when
you want spwn to ignore (say) codex OAuth on your machine but leave
the codex CLI working.
```

```text
spwn auth disable <provider>
```

## spwn auth enable

Reverse a previous `disable`.

```text
spwn auth enable <provider>
```

## spwn auth login

Set up credentials for a provider.

```text
Register credentials for a provider. The simplest path is an
API key:
```

```text
  spwn auth login anthropic --api-key sk-ant-...
```

```text
For OAuth-backed subscription access (Claude.ai / ChatGPT Plus via codex),
run the upstream CLI login first, then re-run this command — spwn will
detect the new credential and record it:
```

```text
  claude login   # then: spwn auth login anthropic
  codex login    # then: spwn auth login openai
```

```text
For hosted MCP providers (Notion, …), the command runs the OAuth dance
itself in a one-shot helper container — click the link it prints,
approve in your browser, the token persists for every world:
```

```text
  spwn auth login notion
```

```text
spwn auth login <provider> [flags]
```

```text
      --api-key string   Save an API key for this provider
```

## spwn auth logout

Clear stored credentials for a provider.

```text
Remove every stored credential for a provider — cache file,
macOS keychain entry, runtime-CLI auth files. Does NOT unset env vars
(the shell owns those); any active env vars are surfaced so you know
to unset them manually.
```

```text
  spwn auth logout anthropic
  spwn auth logout openai
  spwn auth logout anthropic --method api_key   # spare keychain
```

```text
spwn auth logout <provider> [flags]
```

```text
      --method string   Scope logout to a single method (oauth | api_key)
```

## spwn auth use

Pick which credential method spwn should prefer.

```text
Flip the active method for a provider. Run without a method
to clear the preference (back to auto-select).
```

```text
Example:
  spwn auth use anthropic oauth
  spwn auth use openai api_key
```

```text
spwn auth use <provider> <method>
```

## spwn automation

Trigger-driven agent wakeups (cron + filesystem).

```text
Automations bind a trigger (cron expression or filesystem watch)
to one of your project's agents. The architect daemon fires them as
events arrive and writes a receipt for every dispatch.
```

```text
Declare automations under spwn.yaml#worlds.<name>.automations. See
docs/guides/automations.md for the full schema.
```

- [spwn automation daemon](#spwn-automation-daemon) — Run the engine for the current project until interrupted.
- [spwn automation logs](#spwn-automation-logs) — Tail .spwn/runs.jsonl receipts.
- [spwn automation ls](#spwn-automation-ls) — List every automation declared in spwn.yaml.
- [spwn automation status](#spwn-automation-status) — Engine state and last-fired snapshot.

## spwn automation daemon

Run the engine for the current project until interrupted.

```text
Loads the project's automations from spwn.yaml, registers them
with the engine, and blocks. Triggers fire as configured (cron
expressions evaluated against the host clock; filesystem watches via
fsnotify). Each fire writes a receipt to .spwn/runs.jsonl.
```

```text
Stop with Ctrl-C — the engine drains in-flight dispatches before
exiting.
```

```text
Catch-up: on startup, every cron automation that fired before is
checked for missed slots. With catchup: collapse (the default), one
fire is dispatched on resume regardless of how many slots elapsed
during downtime, with the missed count exposed to the prompt
template via {{ .Missed }}. catchup: skip drops missed slots and
resumes at the next scheduled time.
```

```text
spwn automation daemon
```

## spwn automation logs

Tail .spwn/runs.jsonl receipts.

```text
spwn automation logs [flags]
```

```text
  -f, --follow      stream new receipts as they arrive
  -n, --limit int   show at most N most-recent receipts before --follow takes over (default 50)
```

## spwn automation ls

List every automation declared in spwn.yaml.

```text
spwn automation ls
```

## spwn automation status

Engine state and last-fired snapshot.

```text
spwn automation status
```

## spwn build

Transpile the project and compile it into a Docker image.

```text
Transpile the project with the target runtime (default: claude-code)
and compile the result into a derived Docker compile.
```

```text
The image is FROM spwn-world:latest by default, with the transpiled
tree COPY'd to /world/. The resulting image carries the project's
name and the runtime name as Docker labels, so it's push-ready and
reproducible.
```

```text
Pass --tree-only to stop after the transpile step and write the
generated file tree to --output (default: ./dist). No Docker
required, useful for previewing renderer output or authoring a
new runtime backend.
```

```text
Use 'spwn up' to spawn a world from the current project. Use 'spwn
check --deep' to run the transpile dry-run as part of validation.
```

```text
Examples:
  spwn build                                  # transpile + image, tag spwn-<project>:latest
  spwn build --tag spwn-myproj:v1
  spwn build --base spwn-world:2.1
  spwn build --runtime claude-code
  spwn build --world <name>                   # multi-world projects
  spwn build --no-cache
  spwn build --json
  spwn build --tree-only                      # transpile only, write to ./dist
  spwn build --tree-only --output ./preview
  spwn build --tree-only --dry-run            # list paths, touch nothing
  spwn build --tree-only --agent neo          # filter to one agent
```

```text
spwn build [flags]
```

```text
      --agent string     Compile only the named agent (tree-only mode)
      --base string      Base image to derive from (default: $SPWN_BASE_IMAGE, else spwn-world:latest)
      --dry-run          Print paths that would be written, don't touch disk (requires --tree-only)
      --force            Overwrite existing files in --output without prompting (requires --tree-only)
      --json             Emit a machine-readable build report on stdout
      --no-cache         Disable Docker build cache
  -o, --output string    Output directory for --tree-only mode (default "dist")
      --runtime string   Target runtime. Defaults to the runtime declared in agent.yaml (fallback: claude-code)
      --tag string       Image tag (default: spwn-<project>:latest)
      --tree-only        Stop after the compile step; write the Tree to --output instead of building a Docker image
      --world string     World from spwn.yaml to build (required for multi-world projects)
```

## spwn check

Validate the project tree against spwn.yaml.

```text
Walks up from the current directory looking for spwn.yaml, then runs
every validation rule against the project. Reports issues grouped by
severity. Exits non-zero when errors are found (or warnings, with
--strict).
```

```text
spwn check [flags]
```

```text
      --deep     Also run a compile dry-run and report renderer errors
      --json     Emit results as structured JSON on stdout
      --strict   Exit non-zero on warnings, not just errors
```

## spwn cookie-sync

Browser extension that auto-syncs session cookies to the gate (status + providers).

```text
Browser extension that auto-syncs session cookies to the gate.
```

```text
The extension watches your normal browser sessions on sites the gate
knows about (X today; LinkedIn etc. as elements are added) and pushes
the relevant session cookies to a locally-running spwn-gate. No
pairing, no secret — the gate listens on 127.0.0.1 only and accepts
just the cookie names each element declared, so other local processes
can't sneak unrelated cookies in.
```

```text
Setup is two steps:
```

```text
  1. spwn gate start                     # if not already running
  2. Open chrome://extensions/ → Developer mode → Load unpacked →
     select apps/spwn-cookie-sync/ in this repo
```

```text
Then browse normally. The popup shows ● connected / ○ pending per
provider in real-time.
```

```text
spwn cookie-sync
```

- [spwn cookie-sync providers](#spwn-cookie-sync-providers) — List the providers the gate accepts cookie syncs for.
- [spwn cookie-sync status](#spwn-cookie-sync-status) — Show registered providers and per-provider last-sync timestamps.

## spwn cookie-sync providers

List the providers the gate accepts cookie syncs for.

```text
spwn cookie-sync providers
```

## spwn cookie-sync status

Show registered providers and per-provider last-sync timestamps.

```text
spwn cookie-sync status
```

## spwn down

Destroy a world.

```text
spwn down [name-or-id] [flags]
```

```text
      --all   Destroy all running worlds
```

## spwn gate

Manage the host-side credential broker (start/stop/status/logs/restart).

```text
Manage the host-side credential broker.
```

```text
The gate is a long-running container that holds OAuth credentials,
hosts upstream MCP servers, and exposes them to world containers as
authenticated MCP endpoints. World containers never see credentials —
they get tiny CLI wrappers that route through the gate.
```

```text
Auto-started on first `spwn up`; explicit lifecycle commands let you
inspect, restart, or troubleshoot.
```

- [spwn gate logs](#spwn-gate-logs) — Stream gate logs (docker logs).
- [spwn gate restart](#spwn-gate-restart) — Stop + start the gate container; --rebuild forces a fresh image build first.
- [spwn gate start](#spwn-gate-start) — Build the gate image (if missing) and start the container.
- [spwn gate status](#spwn-gate-status) — Show whether the gate is running.
- [spwn gate stop](#spwn-gate-stop) — Stop and remove the gate container (image stays on disk).

## spwn gate logs

Stream gate logs (docker logs).

```text
spwn gate logs [flags]
```

```text
  -f, --follow     Follow log output
  -n, --tail int   Number of lines from the tail (0 = all) (default 50)
```

## spwn gate restart

Stop + start the gate container; --rebuild forces a fresh image build first.

```text
spwn gate restart [flags]
```

```text
      --rebuild   Force a fresh image build before restart
```

## spwn gate start

Build the gate image (if missing) and start the container.

```text
spwn gate start [flags]
```

```text
      --rebuild   Force a fresh image build (use after a spwn binary upgrade)
```

## spwn gate status

Show whether the gate is running.

```text
spwn gate status
```

## spwn gate stop

Stop and remove the gate container (image stays on disk).

```text
spwn gate stop
```

## spwn init

Scaffold a spwn project in the current directory.

```text
Scaffold a spwn project in the current directory.
```

```text
Without arguments, creates a blank spwn.yaml plus a default ./spwn/
tree (one world, one agent) and adds .spwn/ to .gitignore.
```

```text
A positional example ref installs one of the bundled gallery entries
into the current directory. Bare names resolve through the catalog:
```

```text
    spwn init matrix          # shorthand for spwn init spwn:matrix
    spwn init spwn:matrix     # explicit form
```

```text
Use --global to instead seed ~/.spwn/ with a world config (legacy
user-home mode, kept for backward compatibility).
```

```text
spwn init [example-ref] [flags]
```

```text
      --backend string   Pin the scaffolded agent to a runtime backend (e.g. claude-code, codex). Leave empty to let spwn resolve at spawn time.
  -f, --force            Overwrite existing spwn.yaml
      --global           Initialise ~/.spwn/ (legacy user-home mode)
      --name string      Project name (default: current directory name)
```

## spwn inspect

Show per-agent composition: deps, skills, hooks.

```text
Inspect a spwn project: one block per agent, showing the
resolved dependency tree (with transitive (*)-dedup and composition
badges), the skills contributed by tool deps, and the hooks bound
to the agent.
```

```text
Mirrors the kubectl describe / cargo tree convention: key-value
header, section titles with counts, whitespace-separated sections.
```

```text
Examples:
  spwn inspect            # every agent
  spwn inspect neo        # one agent
  spwn inspect --offline  # skip live world-status lookup
```

```text
spwn inspect [agent] [flags]
```

```text
      --offline   Skip live world-status lookup (faster, no Docker calls)
```

## spwn install

Install a dependency into the project.

```text
Add a catalog, GitHub, or local dependency to agent manifests and pin it in spwn.lock.
```

```text
Bare names resolve to the spwn: catalog ("spwn install qmd" installs spwn:qmd).
Without --agent, the ref is added to every agent in the project.
Local refs (skill/, tool/, hook/) require --agent to pick a target.
```

```text
Examples:
  spwn install python                         # catalog dep, every agent
  spwn install spwn:python                    # explicit form, every agent
  spwn install github:jterrazz/research-skills
  spwn install qmd --agent mark               # catalog dep, only mark
  spwn install skill/refine --agent dylan     # local skill, only dylan
```

```text
spwn install <ref> [flags]
```

```text
      --agent string   Target a single agent instead of every agent in the project
```

## spwn logs

Show the system event log across worlds and agents.

```text
Show the spwn event log - spawned worlds, created agents, dream cycles,
snapshots, messages, and every other discrete thing that happened.
```

```text
Scope it with --world or --agent, or use the per-entity shortcuts:
  spwn world logs <id>        events for one world
  spwn agent logs <name>      events for one agent
  spwn architect logs         architect daemon events
```

```text
spwn logs [flags]
```

```text
  -a, --agent string                 Filter by agent name
  -n, --limit int                    Number of events to show (default 20)
  -t, --type string                  Filter by event type (e.g. agent.dreamed)
  -w, --world spwn world logs <id>   Filter by world config name (e.g. 'matrix' from spwn.yaml#worlds). Use spwn world logs <id> to filter by a runtime world ID.
```

## spwn ls

Agent-centric status (running / stopped / orphan).

```text
spwn ls [flags]
```

```text
      --json           Emit results as structured JSON on stdout
      --world string   Filter agents by world ID
```

## spwn skill

Author bare-markdown skill files.

```text
Skills are procedures, playbooks, or pieces of knowledge authored as
bare markdown files at spwn/skills/<name>.md. Attach a skill to an
agent with:
  spwn agent add <agent> --dep <skill-name>
```

- [spwn skill edit](#spwn-skill-edit) — [experimental] Open a skill file in $EDITOR.
- [spwn skill ls](#spwn-skill-ls) — List skill files.
- [spwn skill new](#spwn-skill-new) — Author a new skill file.
- [spwn skill rm](#spwn-skill-rm) — Delete a skill file.
- [spwn skill show](#spwn-skill-show) — Display a skill file.

## spwn skill edit

[experimental] Open a skill file in $EDITOR.

```text
[experimental] Open a skill file in $EDITOR
```

```text
⚠ Experimental: this command is in development and may change or break without notice.
```

```text
spwn skill edit <skill-name>
```

## spwn skill ls

List skill files.

```text
spwn skill ls
```

## spwn skill new

Author a new skill file.

```text
spwn skill new <skill-name>
```

## spwn skill rm

Delete a skill file.

```text
spwn skill rm <skill-name>
```

## spwn skill show

Display a skill file.

```text
spwn skill show <skill-name>
```

## spwn status

Show the full status of your spwn environment.

```text
spwn status
```

## spwn uninstall

Remove a dependency from the project.

```text
Remove a dependency from agent manifests. When no agent still carries the
ref, the lockfile pin is dropped too.
```

```text
Without --agent, the ref is removed from every agent. Pass --agent <name>
to detach it from a single agent while leaving others untouched.
```

```text
Examples:
  spwn uninstall python                     # every agent
  spwn uninstall skill/refine --agent mark  # only mark
```

```text
spwn uninstall <ref> [flags]
```

```text
      --agent string   Target a single agent instead of every agent in the project
```

## spwn up

Spawn a world - an isolated reality for agents.

```text
Spawn a world - the Big Bang.
```

```text
Inside a spwn project:
  spwn up             brings up every world declared in spwn.yaml
  spwn up <name>      brings up a specific world from spwn.yaml
```

```text
Outside a project, the legacy global-mode flags still work and spawn
a one-off world from ~/.spwn/worlds/<config>.yaml.
```

```text
spwn up [name] [flags]
```

```text
  -a, --agent stringArray       Agent name (repeatable; first agent becomes chief in multi-agent worlds)
      --backend string          Override the runtime backend for this spawn (e.g. claude-code, codex). Skips the auth-state auto-resolver.
  -c, --config string           Named world config (default: default)
      --force-rebuild           Ignore the image cache and rebuild the world image from scratch
  -i, --interactive             Drop into the agent's session after spawn
  -n, --name string             Display name for the world
  -w, --workspace stringArray   Host directory to mount. Repeatable. Forms: "path", "name=path", "name=path:ro". Omit for ephemeral.
  -u, --world string            Explicit path to a YAML config file
```

## spwn upgrade

Upgrade spwn to the latest version.

```text
Downloads and installs the latest spwn release from GitHub.
```

```text
Fetches the release binary for your OS/architecture, verifies its SHA256
against the checksums published with the release, and atomically replaces
the current binary. Running worlds are stopped gracefully before the swap.
```

```text
spwn upgrade [flags]
```

```text
      --channel string   Release channel: stable or beta (default "stable")
      --check            Check for updates but do not install
      --force            Install even if already up to date
```

## spwn world

Manage worlds - ephemeral runtime instances for agents.

```text
spwn world [name] [flags]
```

```text
  -a, --agent stringArray       Agent name (repeatable; first agent becomes chief in multi-agent worlds)
      --backend string          Override the runtime backend for this spawn (e.g. claude-code, codex). Skips the auth-state auto-resolver.
  -c, --config string           Named world config (default: default)
      --force-rebuild           Ignore the image cache and rebuild the world image from scratch
  -i, --interactive             Drop into the agent's session after spawn
  -n, --name string             Display name for the world
  -w, --workspace stringArray   Host directory to mount. Repeatable. Forms: "path", "name=path", "name=path:ro". Omit for ephemeral.
  -u, --world string            Explicit path to a YAML config file
```

- [spwn world create](#spwn-world-create) — Declare a new world in spwn.yaml.
- [spwn world destroy](#spwn-world-destroy) — Destroy a world.
- [spwn world enter](#spwn-world-enter) — Open an interactive shell inside a running world.
- [spwn world inspect](#spwn-world-inspect) — Inspect a running world - agents, workspaces, status.
- [spwn world knowledge](#spwn-world-knowledge) — Read a world's shared knowledge.
- [spwn world list](#spwn-world-list) — List declared worlds and their running status.
- [spwn world logs](#spwn-world-logs) — Show the event log for a specific world.
- [spwn world rename](#spwn-world-rename) — Rename a world (omit name to clear).
- [spwn world rm](#spwn-world-rm) — Remove a world declaration from spwn.yaml.
- [spwn world snap](#spwn-world-snap) — World snapshots - save, ls, restore, rm.
- [spwn world start](#spwn-world-start) — Start a world (alias for `spwn up`).
- [spwn world stop](#spwn-world-stop) — Stop a world (alias for `spwn down`).
- [spwn world up](#spwn-world-up) — Spawn a world - an isolated reality for agents.

## spwn world create

Declare a new world in spwn.yaml.

```text
Append a worlds.<name> entry to the project's spwn.yaml.
```

```text
This is a pure config write - no container is created. Once declared,
spawn the world with "spwn world start <name>" or "spwn world <name>".
```

```text
The agents listed via --agent must already exist on disk under
spwn/agents/<name>/. Use "spwn agent create" first if they don't.
```

```text
spwn world create <name> [flags]
```

```text
  -a, --agent stringArray       Agent name (repeatable). Must already exist under spwn/agents/
  -w, --workspace stringArray   Workspace mount. Forms: "path", "name=path", "name=path:ro"
```

## spwn world destroy

Destroy a world.

```text
spwn world destroy [world-id] [flags]
```

```text
      --all   Destroy all running worlds
```

## spwn world enter

Open an interactive shell inside a running world.

```text
spwn world enter <world-id>
```

## spwn world inspect

Inspect a running world - agents, workspaces, status.

```text
spwn world inspect <world-id>
```

## spwn world knowledge

Read a world's shared knowledge.

```text
Each world carries its own knowledge base at /world/knowledge/ inside the
container - shared notes, decisions, and context for the agents working there.
Knowledge is per-world; destroy the world and its knowledge goes with it.
```

- [spwn world knowledge ls](#spwn-world-knowledge-ls) — List all knowledge files in a world.
- [spwn world knowledge show](#spwn-world-knowledge-show) — Show the contents of a knowledge file.

## spwn world knowledge ls

List all knowledge files in a world.

```text
spwn world knowledge ls <world-id>
```

## spwn world knowledge show

Show the contents of a knowledge file.

```text
spwn world knowledge show <world-id> <path>
```

## spwn world list

List declared worlds and their running status.

```text
spwn world list [flags]
```

```text
      --json   Emit results as structured JSON on stdout
```

## spwn world logs

Show the event log for a specific world.

```text
spwn world logs <world-id> [flags]
```

```text
  -n, --limit int   Number of events to show (default 50)
```

## spwn world rename

Rename a world (omit name to clear).

```text
spwn world rename <world-id> [name]
```

## spwn world rm

Remove a world declaration from spwn.yaml.

```text
Remove the worlds.<name> entry from spwn.yaml.
```

```text
This only edits config - it does NOT stop a running container. If
the world is currently running, stop it first with "spwn world stop
<name>".
```

```text
The agents listed by the world stay on disk; their minds are
preserved. Other worlds may still reference them.
```

```text
spwn world rm <name>
```

## spwn world snap

World snapshots - save, ls, restore, rm.

```text
Save, list, restore, and remove world snapshots.
```

```text
spwn world snap
```

- [spwn world snap ls](#spwn-world-snap-ls) — List all snapshots.
- [spwn world snap restore](#spwn-world-snap-restore) — Restore a world from a snapshot.
- [spwn world snap rm](#spwn-world-snap-rm) — Remove a snapshot.
- [spwn world snap save](#spwn-world-snap-save) — Save world state as a snapshot.

## spwn world snap ls

List all snapshots.

```text
spwn world snap ls
```

## spwn world snap restore

Restore a world from a snapshot.

```text
Creates a new world from a previously saved snapshot. The snapshot format is: w-{id}--{name}
```

```text
spwn world snap restore <snapshot> [flags]
```

```text
  -a, --agent string            Agent name (omit for an empty world)
  -c, --config string           Named world config (default: default)
  -w, --workspace stringArray   Host dir to mount. Repeatable: "path", "name=path", "name=path:ro". Omit for ephemeral.
```

## spwn world snap rm

Remove a snapshot.

```text
spwn world snap rm <snapshot>
```

## spwn world snap save

Save world state as a snapshot.

```text
spwn world snap save <world-id> [flags]
```

```text
      --name string   Name for the snapshot
```

## spwn world start

Start a world (alias for `spwn up`).

```text
spwn world start [name] [flags]
```

```text
  -a, --agent stringArray       Agent name (repeatable; first agent becomes chief in multi-agent worlds)
      --backend string          Override the runtime backend for this spawn (e.g. claude-code, codex). Skips the auth-state auto-resolver.
  -c, --config string           Named world config (default: default)
      --force-rebuild           Ignore the image cache and rebuild the world image from scratch
  -i, --interactive             Drop into the agent's session after spawn
  -n, --name string             Display name for the world
  -w, --workspace stringArray   Host directory to mount. Repeatable. Forms: "path", "name=path", "name=path:ro". Omit for ephemeral.
  -u, --world string            Explicit path to a YAML config file
```

## spwn world stop

Stop a world (alias for `spwn down`).

```text
spwn world stop [name-or-id] [flags]
```

```text
      --all   Destroy all running worlds
```

## spwn world up

Spawn a world - an isolated reality for agents.

```text
Spawn a world - the Big Bang.
```

```text
Inside a spwn project:
  spwn up             brings up every world declared in spwn.yaml
  spwn up <name>      brings up a specific world from spwn.yaml
```

```text
Outside a project, the legacy global-mode flags still work and spawn
a one-off world from ~/.spwn/worlds/<config>.yaml.
```

```text
spwn world up [name] [flags]
```

```text
  -a, --agent stringArray       Agent name (repeatable; first agent becomes chief in multi-agent worlds)
      --backend string          Override the runtime backend for this spawn (e.g. claude-code, codex). Skips the auth-state auto-resolver.
  -c, --config string           Named world config (default: default)
      --force-rebuild           Ignore the image cache and rebuild the world image from scratch
  -i, --interactive             Drop into the agent's session after spawn
  -n, --name string             Display name for the world
  -w, --workspace stringArray   Host directory to mount. Repeatable. Forms: "path", "name=path", "name=path:ro". Omit for ephemeral.
  -u, --world string            Explicit path to a YAML config file
```
