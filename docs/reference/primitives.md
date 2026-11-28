# Primitives

Everything an agent is made of is a declarative file, reviewed in PRs and pinned in lockfiles. This chapter is the canonical reference for those blocks and the grammar that references them. The README carries the marketing tour of the same ideas; this is the working spec.

## The project on disk

A Spwn project is per-repo: everything an agent is made of is committed beside the code it works on. `~/.spwn/` holds only user-level credentials and the activity log, and nothing a teammate needs to reproduce the project.

```text
my-project/
├── spwn.yaml       # manifest: version, name, inline worlds map, project-wide deps
├── spwn.lock       # lockfile: pinned catalog deps
├── agents/neo/
│   ├── agent.yaml     # composition: dependencies + runtime.backend
│   ├── AGENTS.md      # provider-neutral entry point (compiled per runtime)
│   ├── SOUL.md        # who the agent is (purpose, voice, values)
│   ├── playbooks/     # promoted patterns (auto-indexed from name:/description: headers)
│   └── journal/       # per-run history
├── skills/         # project-scoped skills   (skill/<name> → skills/<name>.md)
├── tools/          # project-scoped tools    (tool/<name>  → tools/<name>/)
├── hooks/          # project-scoped hooks    (hook/<name>  → hooks/<name>.yaml)
├── commands/       # project-scoped commands (command/<name> → commands/<name>.md)
├── profiles/       # project-scoped profiles
├── knowledge/      # world-scoped facts, bind-mounted at /world/knowledge/ (default path)
└── .spwn/          # gitignored local state
    ├── state.json               # live world IDs bound to this project
    ├── runs.jsonl               # automation receipts (one line per fire)
    ├── automations/state.json   # last-fired cursor per automation (catch-up math)
    └── cache/
```

### The folders sit at the root

The six kinds a project is made of (agents, skills, tools, hooks, commands, profiles) each have one folder at the project's root, the folder that holds `spwn.yaml`. Nothing else at the root is read as the project's: the repository's own code sits beside them.

`folders:` in `spwn.yaml` renames one kind's folder and keeps the others:

```yaml
folders:
    agents: crew
    skills: lib/skills
```

A value is a slash path from the root whose every name is lowercase letters, digits, `-` and `_`. It is never `.`, `..`, a name starting with a dot (so never `.git` or `.spwn`), an absolute path, `~`, a colon or `spwn/`, and no two kinds share a folder or sit one inside the other. A refused entry is `spwn-yaml.folders`, naming its key.

### The old layout is refused

A project whose kinds sat under one `spwn/` folder is the old layout, and every command refuses it before it reads anything else:

```text
spwn/ is the old project layout: its folders now sit at the project's root. Move each one up (mv spwn/agents agents, and the same for skills, tools, hooks, commands, profiles), point knowledge: at its new path, change any .gitignore line naming spwn/, then run again.
```

It is the old layout when the root's `spwn` is a folder holding one of `agents`, `skills`, `tools`, `hooks`, `commands`, `profiles` or `hooks.yaml`, or when `spwn` is a link, which is never followed. When a kind sits both under `spwn/` and at the root, the refusal names those folders to merge. The decision is [ADR-024](../decisions/024-a-projects-folders-sit-at-its-root.md).

```text
~/.spwn/                         # USER-LEVEL only, never per-project
├── credentials/                 # auth material surfaced to containers at /credentials
└── activity.jsonl               # global activity log
```

The two manifests compose. `spwn.yaml` declares project-wide `dependencies:`, an optional `runtime.backend` default, and the inline `worlds:` map; `agent.yaml` declares one agent's `dependencies:` and its `runtime.backend`. An agent's deps are **unioned** with the project-wide pool (it can add to that pool, never remove from it), and the union is exactly what materializes inside that agent's container.

## The dependency grammar

An agent's composition is one `dependencies:` list. The grammar splits **source** (the colon prefix) from **type** (the leading path segment):

| Ref form                | Kind        | Resolves to                                                        |
| ----------------------- | ----------- | ------------------------------------------------------------------ |
| `spwn:<name>`           | catalog dep | bundled catalog entry (e.g. `spwn:unix`, `spwn:git`, `spwn:codex`) |
| `github:<owner>/<repo>` | remote dep  | remote package fetched at resolve time _(planned)_                 |
| `skill/<name>`          | local block | `skills/<name>.md`                                                 |
| `tool/<name>`           | local block | `tools/<name>/`                                                    |
| `hook/<name>`           | local block | `hooks/<name>.yaml`                                                |
| `command/<name>`        | local block | `commands/<name>.md`                                               |

The four local schemes are iso: a path-style ref, selected per agent, resolving to one file or directory on disk. Each agent inherits only the blocks it explicitly subscribes to. Project-wide `dependencies:` in `spwn.yaml` are unioned into every agent: an agent can add but never remove them.

## `spwn.yaml`: the project manifest

The single root file every project has. Declares which worlds exist, which agents they deploy, and the project-wide defaults every agent inherits.

```yaml
version: 1 # required: schema version (always 1 today; check rejects others)
name: my-project # required: appears in world IDs, UI, logs

runtime: # optional: project-wide runtime default
    backend: spwn:claude-code #   agents that omit runtime.backend inherit this

dependencies: # optional: project-wide dep pool (unioned into every agent)
    - spwn:unix
    - spwn:git

worlds: # required: deployable worlds, keyed by name
    matrix:
        agents: [neo] #   required: agent names; each matches agents/<name>/
        workspaces: [.] #   required: host paths mounted at /workspace
        knowledge: ./knowledge #   optional: bind-mounted at /world/knowledge/
```

Worlds are **inline** map entries: the world record (agents, workspaces, tool overrides, `automations:`) lives in YAML, not in separate files. A world optionally owns one filesystem artifact: the directory named by its `knowledge:` key, bind-mounted at `/world/knowledge/`. Omit the key and no mount happens; the agent is never told a knowledge base exists.

## Agents: `agents/<name>/`

An agent is a first-class entity with identity, a role in a world, declared dependencies, and a persistent mind on disk. Everything is optional except `agent.yaml`.

```text
agents/<name>/
  agent.yaml         # required: declarative config
  AGENTS.md          # provider-neutral system-prompt body (compiled per runtime)
  SOUL.md            # the agent's identity (who they are: purpose, voice, values)
  playbooks/         # reusable procedures (name:/description: frontmatter = auto-indexed)
  journal/           # session history (auto-appended by the system)
```

```yaml
# agent.yaml
name: neo # required; must match the directory
description: CI auditor # one-line pitch
role: worker # chief | manager | worker | npc
team: platform # optional grouping

runtime:
    backend: spwn:claude-code # which runtime drives this agent
    model: opus # pinned into .claude/settings.json#model
    provider: anthropic # auth-path hint (anthropic / openai / ...)

dependencies: # unioned with spwn.yaml's project-wide deps
    - spwn:unix
    - spwn:git
```

At build time Spwn renders each agent's entry file (`CLAUDE.md` for Claude Code, `AGENTS.md` for codex) with world-shared context (physics, faculties, roster) plus the agent's role inlined, so the runtime boots fully loaded, with no `@-imports` to chase.

## Tools: `tools/<name>/tool.yaml`

Runnable dependencies: a binary, an install recipe, sometimes a bundle of sidecar files. Every catalog entry and project-local tool uses the same `tool.yaml` shape.

```yaml
name: 'spwn:unix' # required: ref name (matches the dep form)
version: '24.04' # required: semver, distro pin, or "latest"
description: 'Core Unix utils' # required: one-line summary

dependencies: # optional: other tools this needs
    - 'spwn:mcp2cli'

install: # optional: how to install into the image
    packages:
        apt: [bash, curl, jq] #   apt packages (Debian/Ubuntu base)
    commands: #   shell run after package install
        - 'npm install -g @org/pkg'

verify: # optional: smoke-tests at image-build end (non-zero exit fails)
    - command -v bash

gate: # experimental, see 08-gate.md
    cookies: { domains: [x.com], cookies: [auth_token, ct0] }
    mcp: { entry: ['node', 'index.js', 'mcp-serve'] }
```

The compiler unions all deps across a world's agents, topo-sorts, resolves each to a concrete tool, and bakes apt packages + install commands + env + file drops + skills into one world image. Built-in catalog: `spwn:unix`, `spwn:git`, `spwn:node`, `spwn:claude-code`, `spwn:codex`, `spwn:cli`, `spwn:qmd`, `spwn:architect` and more; the full list is in [`15-dependency-catalog.md`](dependency-catalog.md). Tools with a `gate:` block additionally register with the host-side gate ([Gate](../concepts/gate.md)).

## Skills: `skills/<name>/SKILL.md`

Reusable sub-prompts both runtimes auto-discover at startup. Source form is a directory per skill with an entry at `SKILL.md` and any sidecar files alongside; frontmatter needs at least `name:` and `description:`. The legacy bare form `skills/<name>.md` auto-wraps into `<name>/SKILL.md` on load.

The compiler emits `.claude/skills/<skill>/` (Claude Code) or `.agents/skills/<skill>/` (codex, following the cross-vendor `AGENTS.md` convention, **not** `.codex/skills/`). Tool-shipped skills merge into the same tree.

## Hooks: `hooks/<name>.yaml` (experimental)

Fire on runtime events inside the container. One file = one hook; each agent inherits only the hooks it lists via `hook/<name>`.

```yaml
# hooks/bash-audit.yaml
event: PreToolUse # required: runtime event name
matcher: Bash # optional: scope pattern (defaults to *)
command: echo "[audit] $CLAUDE_TOOL_INPUT" # required: shell fragment
```

Event support differs by runtime: `PreToolUse`, `PostToolUse`, `UserPromptSubmit`, `SessionStart`, `Stop` work on both Claude Code and codex; `Notification`, `SubagentStop`, `PreCompact`, `SessionEnd` are Claude-only. `station project plan` warns when a hook targets an event the selecting agent's runtime doesn't fire, and flags hook files no agent subscribes to. The runtime event registries live in [`../packages/runtimes/<runtime>/events.go`](../../packages/runtimes). YAML source (vs the JSON both runtimes store) buys comments and multi-line commands; the translation from YAML to JSON is the point of `station project apply`.

## Commands: `commands/<name>.md`

Slash-invoked prompt shortcuts. Type `/<name>` and the runtime injects the file body as the next prompt. Iso with the other local blocks; selected per-agent via `command/<name>`.

```markdown
<!-- commands/refactor.md -->

---

description: Refactor the selected code while preserving behaviour.
---

Refactor the code I've selected without changing observable behaviour...
```

The body is written verbatim to `.claude/commands/<name>.md` or `.codex/commands/<name>.md`; frontmatter is interpreted by the runtime, not spwn. Use commands for short prompt shortcuts (5 to 20 lines); use skills for multi-phase capabilities with state and tool plumbing.

## Invariants

- **Input, not output.** The manifests declare what reality should be; what the agent reads at startup (physics, faculties, roster) is what it is after the build ([Physics](../concepts/physics.md)). Operators author YAML, agents read rendered markdown.
- **Declarative all the way down.** No manifest triggers behaviour by being edited; reality changes only through `station project apply` and `spwn up`.
- **Composition, not restriction.** Security comes from what the world image lacks, never from a rule in YAML.
- **Blocks are files.** Every skill, tool, hook, and command is one file or directory on disk, diffable and reviewable like code.
- **Declared, then proven.** A tool's `verify:` probes run at the end of the image build, so a world that is missing what it promised fails the build instead of failing an agent.

## Related

- [CLI](cli.md): `spwn install` / `uninstall` for these refs.
- [`15-dependency-catalog.md`](dependency-catalog.md): the built-in `spwn:*` catalog.
- [Gate](../concepts/gate.md): the `gate:` block and cookie-bearing tools.
