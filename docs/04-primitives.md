# Primitives

Everything an agent is made of is a declarative file, reviewed in PRs and pinned in lockfiles. This chapter is the canonical reference for those blocks and the grammar that references them. The README carries the marketing tour of the same ideas; this is the working spec.

## The dependency grammar

An agent's composition is one `dependencies:` list. The grammar splits **source** (the colon prefix) from **type** (the leading path segment):

| Ref form | Kind | Resolves to |
| -------- | ---- | ----------- |
| `spwn:<name>` | catalog dep | bundled catalog entry (e.g. `spwn:unix`, `spwn:git`, `spwn:codex`) |
| `github:<owner>/<repo>` | remote dep | remote package fetched at resolve time *(planned)* |
| `skill/<name>` | local block | `spwn/skills/<name>.md` |
| `tool/<name>` | local block | `spwn/tools/<name>/` |
| `hook/<name>` | local block | `spwn/hooks/<name>.yaml` |
| `command/<name>` | local block | `spwn/commands/<name>.md` |

The four local schemes are iso: a path-style ref, selected per agent, resolving to one file or directory on disk. Each agent inherits only the blocks it explicitly subscribes to. Project-wide `dependencies:` in `spwn.yaml` are unioned into every agent — an agent can add but never remove them.

## `spwn.yaml` — the project manifest

The single root file every project has. Declares which worlds exist, which agents they deploy, and the project-wide defaults every agent inherits.

```yaml
version: 1                      # required: schema version (always 1 today; check rejects others)
name: my-project                # required: appears in world IDs, UI, logs

runtime:                        # optional: project-wide runtime default
  backend: spwn:claude-code     #   agents that omit runtime.backend inherit this

dependencies:                   # optional: project-wide dep pool (unioned into every agent)
  - spwn:unix
  - spwn:git

worlds:                         # required: deployable worlds, keyed by name
  matrix:
    agents: [neo]               #   required: agent names; each matches spwn/agents/<name>/
    workspaces: [.]             #   required: host paths mounted at /workspace
    knowledge: ./spwn/knowledge #   optional: bind-mounted at /world/knowledge/
```

Worlds are **inline** map entries — the world record (agents, workspaces, tool overrides, `automations:`) lives in YAML, not in separate files. A world optionally owns one filesystem artifact: the directory named by its `knowledge:` key, bind-mounted at `/world/knowledge/`. Omit the key and no mount happens — the agent is never told a knowledge base exists.

## Agents — `spwn/agents/<name>/`

An agent is a first-class entity with identity, a role in a world, declared dependencies, and a persistent mind on disk. Everything is optional except `agent.yaml`.

```
spwn/agents/<name>/
  agent.yaml         # required: declarative config
  AGENTS.md          # provider-neutral system-prompt body (compiled per runtime)
  SOUL.md            # the agent's identity (who they are — purpose, voice, values)
  playbooks/         # reusable procedures (name:/description: frontmatter = auto-indexed)
  journal/           # session history (auto-appended by the system)
```

```yaml
# agent.yaml
name: neo                     # required; must match the directory
description: CI auditor       # one-line pitch
role: worker                  # chief | manager | worker | npc
team: platform                # optional grouping

runtime:
  backend: spwn:claude-code   # which runtime drives this agent
  model: opus                 # pinned into .claude/settings.json#model
  provider: anthropic         # auth-path hint (anthropic / openai / …)

dependencies:                 # unioned with spwn.yaml's project-wide deps
  - spwn:unix
  - spwn:git
```

At build time spwn renders each agent's entry file (`CLAUDE.md` for Claude Code, `AGENTS.md` for codex) with world-shared context (physics, faculties, roster) plus the agent's role inlined, so the runtime boots fully loaded — no `@-imports` to chase.

## Tools — `spwn/tools/<name>/tool.yaml`

Runnable dependencies: a binary, an install recipe, sometimes a bundle of sidecar files. Every catalog entry and project-local tool uses the same `tool.yaml` shape.

```yaml
name: "spwn:unix"               # required: ref name (matches the dep form)
version: "24.04"                # required: semver, distro pin, or "latest"
description: "Core Unix utils"  # required: one-line summary

dependencies:                   # optional: other tools this needs
  - "spwn:mcp2cli"

install:                        # optional: how to install into the image
  packages:
    apt: [bash, curl, jq]       #   apt packages (Debian/Ubuntu base)
  commands:                     #   shell run after package install
    - "npm install -g @org/pkg"

verify:                         # optional: smoke-tests at image-build end (non-zero exit fails)
  - command -v bash

gate:                           # 🚧 experimental — see 06-gate.md
  cookies: { domains: [x.com], cookies: [auth_token, ct0] }
  mcp: { entry: ["node", "index.js", "mcp-serve"] }
```

The compiler unions all deps across a world's agents, topo-sorts, resolves each to a concrete tool, and bakes apt packages + install commands + env + file drops + skills into one world image. Built-in catalog: `spwn:unix`, `spwn:git`, `spwn:node`, `spwn:claude-code`, `spwn:codex`, `spwn:cli`, `spwn:qmd`, `spwn:architect`, … — the full list is in [`dependency-catalog.md`](dependency-catalog.md). Tools with a `gate:` block additionally register with the host-side gate ([Gate](06-gate.md)).

## Skills — `spwn/skills/<name>/SKILL.md`

Reusable sub-prompts both runtimes auto-discover at startup. Source form is a directory per skill with an entry at `SKILL.md` and any sidecar files alongside; frontmatter needs at least `name:` and `description:`. The legacy bare form `spwn/skills/<name>.md` auto-wraps into `<name>/SKILL.md` on load.

The compiler emits `.claude/skills/<skill>/` (Claude Code) or `.agents/skills/<skill>/` (codex — the cross-vendor `AGENTS.md` convention, **not** `.codex/skills/`). Tool-shipped skills merge into the same tree.

## Hooks — `spwn/hooks/<name>.yaml` 🚧 experimental

Fire on runtime events inside the container. One file = one hook; each agent inherits only the hooks it lists via `hook/<name>`.

```yaml
# spwn/hooks/bash-audit.yaml
event: PreToolUse            # required: runtime event name
matcher: Bash               # optional: scope pattern (defaults to *)
command: echo "[audit] $CLAUDE_TOOL_INPUT"   # required: shell fragment
```

Event support differs by runtime — `PreToolUse`, `PostToolUse`, `UserPromptSubmit`, `SessionStart`, `Stop` work on both Claude Code and codex; `Notification`, `SubagentStop`, `PreCompact`, `SessionEnd` are Claude-only. `spwn check` warns when a hook targets an event the selecting agent's runtime doesn't fire, and flags hook files no agent subscribes to. The runtime event registries live in [`../packages/runtimes/<runtime>/events.go`](../packages/runtimes). YAML source (vs the JSON both runtimes store) buys comments and multi-line commands; the YAML→JSON translation is the point of `spwn build`.

## Commands — `spwn/commands/<name>.md`

Slash-invoked prompt shortcuts. Type `/<name>` and the runtime injects the file body as the next prompt. Iso with the other local blocks; selected per-agent via `command/<name>`.

```markdown
<!-- spwn/commands/refactor.md -->
---
description: Refactor the selected code while preserving behaviour.
---
Refactor the code I've selected without changing observable behaviour…
```

The body is written verbatim to `.claude/commands/<name>.md` or `.codex/commands/<name>.md`; frontmatter is interpreted by the runtime, not spwn. Use commands for short prompt shortcuts (5–20 lines); use skills for multi-phase capabilities with state and tool plumbing.

## Related

- [Getting started](01-getting-started.md) — the config hierarchy in context.
- [CLI](03-cli.md) — `spwn install` / `uninstall` for these refs.
- [`dependency-catalog.md`](dependency-catalog.md) — the built-in `spwn:*` catalog.
- [Gate](06-gate.md) — the `gate:` block and cookie-bearing tools.
