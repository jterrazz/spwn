# Package Catalog

Spwn worlds are assembled from composable packages. Each package is a self-contained unit declared by a single `package.yaml`: it knows how to install itself, how to verify it works, and optionally ships a skill or injects runtime config. The imagebuilder resolves dependencies, deduplicates packages, and produces one optimized Docker image.

Packages are stackable: `@spwn/qmd` depends on `@spwn/node`, so listing `@spwn/qmd` pulls Node.js in automatically.

Status legend: 🟢 working · 🟡 installed but rough · 🔴 planned.

## SDKs

Language runtimes and core system utilities.

| Package | What it provides | Use when | Status |
|---------|-----------------|----------|--------|
| `@spwn/unix` | bash, coreutils, grep, sed, awk, curl, jq | You need standard shell tools | 🟢 |
| `@spwn/node` | Node.js 20, npm, npx | Your project uses JavaScript/TypeScript | 🟢 |
| `@spwn/python` | Python 3, pip | Your project uses Python | 🟢 |
| `@spwn/build` | make, gcc, g++ | You need to compile C/C++ | 🟢 |

## Runtimes

The thinking engine that drives the agent. Pick one per agent. Runtimes stay in Go (unlike packages) because their spawn-time behavior — credential sync, default config materialisation, prelaunch shell, authentication flows — is too stateful for declarative YAML.

| Runtime | What it provides | Use when | Status |
|---------|-----------------|----------|--------|
| `@spwn/claude-code` | Claude Code CLI + pre-configured auth | You want Anthropic's agent runtime (default) | 🟢 |
| `@spwn/codex` | Codex CLI installed in the image | You want OpenAI-style models | 🟡 installed, no runtime adapter wired |
| `@spwn/aider` | Aider CLI | You want an open-source code-focused runtime | 🔴 planned |

Only `@spwn/claude-code` is wired as a runtime adapter in `packages/world/internal/runtime`. `@spwn/codex` installs the binary and `claude-code` can still shell out to it, but `agent.yaml`'s `runtime:` field only accepts `claude-code` today.

## Tools

Extra capabilities you add to a world. Each ships skills that teach the agent how to use it.

| Package | What it provides | Use when | Status |
|---------|-----------------|----------|--------|
| `@spwn/git` | Git version control | You need source control (almost always) | 🟢 |
| `@spwn/docker-cli` | Docker CLI (DooD) | The agent needs to manage containers | 🟢 |
| `@spwn/qmd` | [QMD](https://github.com/tobi/qmd) on-device search | The agent needs to search docs, notes, or knowledge bases locally | 🟢 |

## Platform

Spwn's own infrastructure. Usually included by default.

| Package | What it provides | Use when | Status |
|---------|-----------------|----------|--------|
| `@spwn/cli` | spwn CLI inside the world | The agent needs to manage its own identity, messages, or sub-worlds | 🟢 |
| `@spwn/architect` | Full orchestration daemon (includes @spwn/cli, @spwn/claude-code, @spwn/docker-cli) | You're running the always-on Architect | 🟡 architect mode is in dev |

## Plugin packages

A **plugin** is a package whose `package.yaml` declares a `plugin:` section. At spawn time the merger reaches into the targeted runtime's config file (e.g. `~/.claude/settings.json`) and shallow-merges the plugin's YAML snippet. That's how MCP servers, shell hooks, or any other runtime-specific wiring show up inside the container without the user having to touch config files.

There is no separate `plugins:` field anywhere — plugins are just packages with richer manifests, installed via `spwn package install @spwn/mempalace` and listed under `agent.yaml#packages:` alongside everything else.

| Package | Targets | What it provides | Status |
|---------|---------|------------------|--------|
| `@spwn/mempalace` | `@spwn/claude-code` | [MemPalace](https://github.com/MemPalace/mempalace) memory palace exposed as an MCP server | 🟡 experimental |

## Package reference kinds

Spwn classifies every package reference in `agent.yaml#packages` (and world-level `packages:`) into one of three kinds:

- **Local** — a bare name like `my-thing`. Resolved against `./spwn/packages/my-thing/` (directory form, full package with its own `package.yaml`) or `./spwn/packages/my-thing.md` (bare-markdown skill). Drop the directory or file and it's picked up automatically.
- **Built-in** — `@spwn/<name>`. Looked up in the catalog shipped with the CLI (see tables above). `spwn check` offers "did you mean X?" hints for typos.
- **Remote registry** — `@<owner>/<name>` with any owner other than `spwn`, e.g. `@jterrazz/python`. Reserved for a future remote registry. Today `spwn check` reports these as `remote registries are not yet supported (ref: …)` so they aren't confused with typos. Until the registry ships, use `@spwn/<name>` or drop a local package under `./spwn/packages/<name>/`.

Catalog refs are pinned in `spwn.lock.yaml` at the project root. Install one with `spwn package install @spwn/<name>` (or the `spwn pkg` alias). `spwn check` flags any drift between agent.yaml and the lockfile.

## Adding your own packages

Every package is described by a `package.yaml` manifest. The schema is small and every field is optional, so a minimal package can be four lines:

```yaml
# spwn/packages/my-thing/package.yaml
name: my-thing
install:
  packages:
    - curl
verify:
  - command -v curl
```

Richer packages can add `commands:`, `user-commands:` (with `{{.Home}}` / `{{.User}}` templating), `files:` (image-path → source-path map), `dependencies:`, `description:`, `plugin:` (with `runtimes:` + `configs:` for runtime-config injection), and optional sibling directories `skills/`, `files/`, `config/`.

Drop the directory under `./spwn/packages/<name>/` to author locally, or under `catalog/packages/<name>/` (inside the spwn monorepo) to ship it in the built-in catalog. The loader picks up both via `go:embed` + filesystem walk — no Go code, no registration list.

For the full schema, see [`packages/image/pkgyaml/schema.go`](../packages/image/pkgyaml/schema.go).
