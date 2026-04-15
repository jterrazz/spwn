# Tool Catalog

Spwn worlds are assembled from composable tools. Each tool is a self-contained plugin: it knows how to install itself, how to verify it works, and what skills to teach the agent. You pick only what you need - the imagebuilder resolves dependencies, deduplicates packages, and produces a single optimized Docker image.

Tools are stackable. `@spwn/qmd` depends on `@spwn/node` - list `@spwn/qmd` and Node.js appears automatically.

Status legend: 🟢 working · 🟡 installed but rough · 🔴 planned.

## SDKs

Language runtimes and core system utilities.

| Tool | What it provides | Use when | Status |
|------|-----------------|----------|--------|
| `@spwn/unix` | bash, coreutils, grep, sed, awk, curl, jq | You need standard shell tools | 🟢 |
| `@spwn/node` | Node.js 20, npm, npx | Your project uses JavaScript/TypeScript | 🟢 |
| `@spwn/python` | Python 3, pip | Your project uses Python | 🟢 |
| `@spwn/build` | make, gcc, g++ | You need to compile C/C++ | 🟢 |

## Runtimes

The thinking engine that drives the agent. Pick one per agent.

| Tool | What it provides | Use when | Status |
|------|-----------------|----------|--------|
| `@spwn/claude-code` | Claude Code CLI + pre-configured auth | You want Anthropic's agent runtime (default) | 🟢 |
| `@spwn/codex` | Codex CLI installed in the image | You want OpenAI-style models | 🟡 binary available, no runtime adapter wired |
| `@spwn/aider` | Aider CLI | You want an open-source code-focused runtime | 🔴 planned |

Only `@spwn/claude-code` is wired as a runtime adapter in `packages/world/internal/runtime`. `@spwn/codex` installs the binary and `claude-code` can still shell out to it, but `agent.yaml`'s `runtime:` field only accepts `claude-code` today.

## Tools

Extra capabilities you add to a world. Each ships skills that teach the agent how to use it.

| Tool | What it provides | Use when | Status |
|------|-----------------|----------|--------|
| `@spwn/git` | Git version control | You need source control (almost always) | 🟢 |
| `@spwn/docker-cli` | Docker CLI (DooD) | The agent needs to manage containers | 🟢 |
| `@spwn/qmd` | [QMD](https://github.com/tobi/qmd) on-device search | The agent needs to search docs, notes, or knowledge bases locally | 🟢 |

## Platform

Spwn's own infrastructure. Usually included by default.

| Tool | What it provides | Use when | Status |
|------|-----------------|----------|--------|
| `@spwn/cli` | spwn CLI inside the world | The agent needs to manage its own identity, messages, or sub-worlds | 🟢 |
| `@spwn/architect` | Full orchestration daemon (includes @spwn/cli, @spwn/claude-code, @spwn/docker-cli) | You're running the always-on Architect | 🟡 architect mode is in dev |

## Plugins

Plugins are tool packs that *also* target a runtime: they reach into the
runtime's config file (e.g. `~/.claude/settings.json`) at spawn time and
merge in their own JSON snippet. That's how MCP servers, shell hooks, or
any other runtime-specific wiring show up inside the container without
the user having to touch config files.

Mechanically, a plugin is a tool whose Go implementation also satisfies
the optional `image.Plugin` interface (`Runtimes()` + `Config(runtime)`).
They live in `catalog/plugins/` and are referenced from `agent.yaml`
under a dedicated `plugins:` field:

```yaml
plugins:
  - "@spwn/mempalace"
```

`plugins:` coexists with `tools:` - both lists are unioned at spawn time
and resolved through the same builder registry, so plugins can depend on
regular tools (and vice-versa).

| Plugin | Targets | What it provides | Status |
|--------|---------|------------------|--------|
| `@spwn/mempalace` | `@spwn/claude-code` | [MemPalace](https://github.com/MemPalace/mempalace) memory palace exposed as an MCP server | 🟡 experimental |

## Tool reference kinds

Spwn classifies every tool reference in `agent.yaml#tools` (and world-level `tools:`) into one of three kinds:

- **Local** - a bare name like `my-tool`. Resolved against `./spwn/tools/<name>/` on disk. Drop a directory there and it's picked up automatically.
- **Built-in** - `@spwn/<name>`. Looked up in the catalog shipped with the CLI (see tables above). `spwn check` offers "did you mean X?" hints for typos.
- **Remote registry** - `@<owner>/<name>` with any owner other than `spwn`, e.g. `@jterrazz/python`. Reserved for a future remote registry. Today `spwn check` reports these as `remote registries are not yet supported (ref: …)` so they aren't confused with typos. Until the registry ships, use `@spwn/<name>` or drop a local pack under `./spwn/tools/<name>/`.

## Adding your own tools

Every tool implements one Go interface:

```go
type Tool interface {
    Name() string           // "@spwn/mytool"
    Kind() Kind             // runtime, tool, sdk, platform
    Version() string        // semver or "latest"
    Dependencies() []string // other tools this requires
    Install() InstallSpec   // apt packages, RUN commands, files
    Verify() []string       // commands that must exit 0
    Skills() fs.FS          // SKILL.md + references (or nil)
}
```

Create a directory under `catalog/tools/mytool/`, implement the interface, add it to `catalog/tools/tools.go`. The test framework validates your tool automatically.
