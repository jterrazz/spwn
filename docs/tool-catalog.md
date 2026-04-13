# Tool Catalog

Spwn worlds are assembled from composable tools. Each tool is a self-contained plugin: it knows how to install itself, how to verify it works, and what skills to teach the agent. You pick only what you need — the imagebuilder resolves dependencies, deduplicates packages, and produces a single optimized Docker image.

Tools are stackable. `@spwn/qmd` depends on `@spwn/node` — list `@spwn/qmd` and Node.js appears automatically.

## SDKs

Language runtimes and core system utilities.

| Tool | What it provides | Use when | Status |
|------|-----------------|----------|--------|
| `@spwn/unix` | bash, coreutils, grep, sed, awk, curl, jq | You need standard shell tools | Available |
| `@spwn/node` | Node.js 20, npm, npx | Your project uses JavaScript/TypeScript | Available |
| `@spwn/python` | Python 3, pip | Your project uses Python | Available |
| `@spwn/build` | make, gcc, g++ | You need to compile C/C++ | Available |

## Runtimes

The thinking engine that drives the agent. Pick one per agent.

| Tool | What it provides | Use when | Status |
|------|-----------------|----------|--------|
| `@spwn/claude-code` | Claude Code CLI + pre-configured auth | You want Anthropic's agent runtime (default) | Available |
| `@spwn/codex` | Codex CLI + pre-configured workspace trust | You want OpenAI models (GPT-5, o3) | Available |
| `@spwn/aider` | Aider CLI | You want an open-source code-focused runtime | Planned |

## Tools

Extra capabilities you add to a world. Each ships skills that teach the agent how to use it.

| Tool | What it provides | Use when | Status |
|------|-----------------|----------|--------|
| `@spwn/git` | Git version control | You need source control (almost always) | Available |
| `@spwn/docker-cli` | Docker CLI (DooD) | The agent needs to manage containers | Available |
| `@spwn/qmd` | [QMD](https://github.com/tobi/qmd) on-device search | The agent needs to search docs, notes, or knowledge bases locally | Available |

## Platform

Spwn's own infrastructure. Usually included by default.

| Tool | What it provides | Use when | Status |
|------|-----------------|----------|--------|
| `@spwn/cli` | spwn CLI inside the world | The agent needs to manage its own identity, messages, or sub-worlds | Available |
| `@spwn/architect` | Full orchestration daemon (includes @spwn/cli, @spwn/claude-code, @spwn/docker-cli) | You're running the always-on Architect | Available |

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

Create a directory under `packages/imagebuilder/catalog/mytool/`, implement the interface, add it to `catalog.go`. The test framework validates your tool automatically.
