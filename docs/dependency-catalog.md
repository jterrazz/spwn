# Dependency Catalog

Spwn worlds are assembled from composable dependencies. Each dependency is a self-contained unit declared by a single `spwn.yaml`: it knows how to install itself, how to verify it works, and optionally ships a skill or injects runtime config. The imagebuilder resolves dependencies, deduplicates them, and produces one optimized Docker image.

Dependencies are stackable: `@spwn/qmd` depends on `@spwn/node`, so listing `@spwn/qmd` pulls Node.js in automatically.

Status legend: 🟢 working · 🟡 installed but rough · 🔴 planned.

## SDKs

Language runtimes and core system utilities.

| Dependency | What it provides | Use when | Status |
|--------|-----------------|----------|--------|
| `@spwn/unix` | bash, coreutils, grep, sed, awk, curl, jq | You need standard shell tools | 🟢 |
| `@spwn/node` | Node.js 20, npm, npx | Your project uses JavaScript/TypeScript | 🟢 |
| `@spwn/python` | Python 3, pip | Your project uses Python | 🟢 |
| `@spwn/build` | make, gcc, g++ | You need to compile C/C++ | 🟢 |

## Runtimes

The thinking engine that drives the agent. Pick one per agent. Runtimes stay in Go (unlike dependencies) because their spawn-time behavior — credential sync, default config materialisation, prelaunch shell, authentication flows — is too stateful for declarative YAML.

| Runtime | What it provides | Use when | Status |
|---------|-----------------|----------|--------|
| `@spwn/claude-code` | Claude Code CLI + pre-configured auth | You want Anthropic's agent runtime (default) | 🟢 |
| `@spwn/codex` | Codex CLI installed in the image | You want OpenAI-style models | 🟡 installed, no runtime adapter wired |
| `@spwn/aider` | Aider CLI | You want an open-source code-focused runtime | 🔴 planned |

Only `@spwn/claude-code` is wired as a runtime adapter in `packages/world/runtime`. `@spwn/codex` installs the binary and `claude-code` can still shell out to it, but `agent.yaml`'s `runtime:` field only accepts `claude-code` today.

## Tools

Extra capabilities you add to a world. Each ships skills that teach the agent how to use it.

| Dependency | What it provides | Use when | Status |
|--------|-----------------|----------|--------|
| `@spwn/git` | Git version control | You need source control (almost always) | 🟢 |
| `@spwn/docker-cli` | Docker CLI (DooD) | The agent needs to manage containers | 🟢 |
| `@spwn/qmd` | [QMD](https://github.com/tobi/qmd) on-device search | The agent needs to search docs, notes, or knowledge bases locally | 🟢 |

## Platform

Spwn's own infrastructure. Usually included by default.

| Dependency | What it provides | Use when | Status |
|--------|-----------------|----------|--------|
| `@spwn/cli` | spwn CLI inside the world | The agent needs to manage its own identity, messages, or sub-worlds | 🟢 |
| `@spwn/architect` | Full orchestration daemon (includes @spwn/cli, @spwn/claude-code, @spwn/docker-cli) | You're running the always-on Architect | 🟡 architect mode is in dev |

## Dependencies with runtime-config injection

Any dependency whose `spwn.yaml` declares a `runtime-config:` section participates in spawn-time config injection. At spawn time the merger reaches into the targeted runtime's config file (e.g. `~/.claude/settings.json`) and shallow-merges the dependency's YAML snippet. That's how MCP servers, shell hooks, or any other runtime-specific wiring show up inside the container without the user having to touch config files.

There is no separate `plugins:` field anywhere — `runtime-config:` is just an optional block on the unified dependency manifest. Install one with `spwn install @spwn/mempalace` and it shows up in `agent.yaml#dependencies:` alongside everything else.

| Dependency | Targets | What it provides | Status |
|--------|---------|------------------|--------|
| `@spwn/mempalace` | `@spwn/claude-code` | [MemPalace](https://github.com/MemPalace/mempalace) memory palace exposed as an MCP server | 🟡 experimental |

## Dependency reference kinds

Spwn classifies every ref in `agent.yaml#dependencies` into one of three kinds:

- **Local** — a bare name like `my-thing`. Resolved against `./spwn/tools/my-thing/` (directory form, full dependency with its own `tool.yaml`) or `./spwn/tools/my-thing.md` (bare-markdown skill). Drop the directory or file and it's picked up automatically.
- **Built-in** — `@spwn/<name>`. Looked up in the catalog shipped with the CLI (see tables above). `spwn check` offers "did you mean X?" hints for typos.
- **Remote registry** — `@<owner>/<name>` with any owner other than `spwn`, e.g. `@jterrazz/python`. Reserved for a future remote registry. Today `spwn check` reports these as `remote registries are not yet supported (ref: …)` so they aren't confused with typos. Until the registry ships, use `@spwn/<name>` or drop a local tool under `./spwn/tools/<name>/`.

Catalog refs are pinned in `spwn.lock` at the project root. Install one with `spwn install @spwn/<name>`. `spwn check` flags any drift between agent.yaml and the lockfile.

## Adding your own dependencies

Every dependency is described by a `tool.yaml` manifest. The schema is small and every field is optional, so a minimal dependency can be four lines:

```yaml
# spwn/tools/my-thing/tool.yaml
name: my-thing
install:
  packages:
    - curl
verify:
  - command -v curl
```

Richer dependencies can add `commands:`, `user-commands:` (with `{{.Home}}` / `{{.User}}` templating), `files:` (image-path → source-path map), `dependencies:`, `description:`, `runtime-config:` (with `runtimes:` + `configs:` for runtime-config injection), and optional sibling directories `skills/`, `files/`, `config/`.

Drop the directory under `./spwn/tools/<name>/` to author locally, or under `catalog/<name>/tools/<name>/` (inside the spwn monorepo) to ship it in the built-in catalog. The loader picks up both via `go:embed` + filesystem walk — no Go code, no registration list.

For the full schema, see [`packages/dependency/schema.go`](../packages/dependency/schema.go).
