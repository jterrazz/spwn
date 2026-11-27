# Commands

Every command of the `spwn` binary, as `spwn <command> --help` prints it. The page is projected from the Cobra command tree by `make docs` and is never edited by hand; the grammar the commands follow is [the CLI](cli.md).

## spwn

spwn - run AI agents in isolated worlds you declare.

```text
The spwn command is Agents, the Spwn app that runs one agent, or a
team of agents, in isolated Docker worlds. Each world has its own
rules (network, filesystem, tools), and each agent keeps its Mind
(its identity and memory) from one world to the next.
```

- [spwn agent](#spwn-agent): Spawn an agent into a world, or open a session with one.
- [spwn auth](#spwn-auth): Manage credentials: dashboard, login, use, logout, disable.
- [spwn automation](#spwn-automation): Trigger-driven agent wakeups (cron + filesystem).
- [spwn down](#spwn-down): Destroy a world.
- [spwn gate](#spwn-gate): Manage the host-side credential broker (start/stop/status/logs/restart).
- [spwn install](#spwn-install): Install a dependency into the project.
- [spwn logs](#spwn-logs): Show the system event log across worlds and agents.
- [spwn uninstall](#spwn-uninstall): Remove a dependency from the project.
- [spwn up](#spwn-up): Spawn a world - an isolated container for agents.
- [spwn upgrade](#spwn-upgrade): Upgrade Spwn to the latest version.
- [spwn world](#spwn-world): Manage worlds - ephemeral runtime instances for agents.

## spwn agent

Spawn an agent into a world, or open a session with one.

```text
Spawn an agent into an existing world.
```

```text
An agent is backed by a Mind - a persistent directory holding its SOUL.md
(purpose, voice, principles), playbooks/, and journal/. The agent survives
after the world is destroyed. Skills are build-time dependencies injected
into /world/skills/; knowledge lives at /world/knowledge/ inside each
world when the manifest opts in via worlds.<name>.knowledge; the path
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

- [spwn agent deploy](#spwn-agent-deploy): Deploy an agent to a running world.
- [spwn agent logs](#spwn-agent-logs): Show the event log for a specific agent.
- [spwn agent talk](#spwn-agent-talk): Talk to a running agent - interactive or one-shot.

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

## spwn agent logs

Show the event log for a specific agent.

```text
spwn agent logs <agent-name> [flags]
```

```text
  -n, --limit int   Number of events to show (default 50)
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

## spwn auth

Manage credentials: dashboard, login, use, logout, disable.

```text
spwn auth
```

- [spwn auth default](#spwn-auth-default): Pick which provider Spwn prefers when multiple are authenticated.
- [spwn auth disable](#spwn-auth-disable): Tell Spwn not to use this provider, even if creds exist.
- [spwn auth enable](#spwn-auth-enable): Reverse a previous `disable`.
- [spwn auth login](#spwn-auth-login): Set up credentials for a provider.
- [spwn auth logout](#spwn-auth-logout): Clear stored credentials for a provider.
- [spwn auth use](#spwn-auth-use): Pick which credential method Spwn should prefer.

## spwn auth default

Pick which provider Spwn prefers when multiple are authenticated.

```text
Set a soft preference for which provider's runtime Spwn picks
when you're logged into more than one and no runtime is pinned at the
project or agent level.
```

```text
This is the durable answer to the "multiple providers authenticated
and no runtime pinned" error. Set it once and Spwn will quietly
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

Tell Spwn not to use this provider, even if creds exist.

```text
Opt a provider out without touching credentials. Useful when
you want Spwn to ignore (say) codex OAuth on your machine but leave
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
run the upstream CLI login first, then re-run this command; Spwn will
detect the new credential and record it:
```

```text
  claude login   # then: spwn auth login anthropic
  codex login    # then: spwn auth login openai
```

```text
For hosted MCP providers (Notion, ...), the command runs the OAuth dance
itself in a one-shot helper container: click the link it prints,
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
Remove every stored credential for a provider: cache file,
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

Pick which credential method Spwn should prefer.

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
to one of your project's agents. The command "spwn automation daemon" fires them as
events arrive and writes a receipt for every dispatch.
```

```text
Declare automations under spwn.yaml#worlds.<name>.automations. See
docs/guides/automations.md for the full schema. "station automations" lists
them and the runs written for them.
```

- [spwn automation daemon](#spwn-automation-daemon): Run the engine for the current project until interrupted.

## spwn automation daemon

Run the engine for the current project until interrupted.

```text
Loads the project's automations from spwn.yaml, registers them
with the engine, and blocks. Triggers fire as configured (cron
expressions evaluated against the host clock; filesystem watches via
fsnotify). Each fire writes a receipt to .spwn/runs.jsonl.
```

```text
Stop with Ctrl-C: the engine drains in-flight dispatches before
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
spwn automation daemon [flags]
```

```text
      --yes   Allow, for this run only, the watch folders outside the project that spwn.yaml names, printing each; never kept
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
authenticated MCP endpoints. World containers never see credentials;
they get tiny CLI wrappers that route through the gate.
```

```text
Auto-started on first `spwn up`; explicit lifecycle commands let you
inspect, restart, or troubleshoot.
```

- [spwn gate logs](#spwn-gate-logs): Stream gate logs (docker logs).
- [spwn gate restart](#spwn-gate-restart): Stop + start the gate container; --rebuild forces a fresh image build first.
- [spwn gate start](#spwn-gate-start): Build the gate image (if missing) and start the container.
- [spwn gate status](#spwn-gate-status): Show whether the gate is running.
- [spwn gate stop](#spwn-gate-stop): Stop and remove the gate container (image stays on disk).

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
Show the Spwn event log - spawned worlds, created agents, dream cycles,
snapshots, messages, and every other discrete thing that happened.
```

```text
Scope it with --world or --agent, or use the per-entity shortcuts:
  spwn world logs <id>        events for one world
  spwn agent logs <name>      events for one agent
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

Spawn a world - an isolated container for agents.

```text
Spawn a world: an isolated container and the agents declared in it.
```

```text
Inside a Spwn project:
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
      --yes                     Allow, for this run only, the host paths outside the project that spwn.yaml mounts, printing each; never kept
```

## spwn upgrade

Upgrade Spwn to the latest version.

```text
Downloads and installs the latest Spwn release from GitHub.
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

- [spwn world create](#spwn-world-create): Declare a new world in spwn.yaml.
- [spwn world enter](#spwn-world-enter): Open an interactive shell inside a running world.
- [spwn world inspect](#spwn-world-inspect): Inspect a running world - agents, workspaces, status.
- [spwn world list](#spwn-world-list): List declared worlds and their running status.
- [spwn world logs](#spwn-world-logs): Show the event log for a specific world.
- [spwn world rename](#spwn-world-rename): Rename a world (omit name to clear).
- [spwn world rm](#spwn-world-rm): Remove a world declaration from spwn.yaml.

## spwn world create

Declare a new world in spwn.yaml.

```text
Append a worlds.<name> entry to the project's spwn.yaml.
```

```text
This is a pure config write - no container is created. Once declared,
spawn the world with "spwn up <name>".
```

```text
The agents listed via --agent must already exist on disk under
spwn/agents/<name>/. Use "station agents create" first if they don't.
```

```text
spwn world create <name> [flags]
```

```text
  -a, --agent stringArray       Agent name (repeatable). Must already exist under spwn/agents/
  -w, --workspace stringArray   Workspace mount. Forms: "path", "name=path", "name=path:ro"
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
the world is currently running, stop it first with "spwn down
<name>".
```

```text
The agents listed by the world stay on disk; their minds are
preserved. Other worlds may still reference them.
```

```text
spwn world rm <name>
```
