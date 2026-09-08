# Dependency catalog

The built-in dependencies shipped with the spwn CLI. Install any of
these with `spwn install spwn:<name>` (or with `--agent <name>` to
scope to one agent). Refs get pinned in `spwn.lock`.

Dependencies are stackable: listing `spwn:qmd` pulls `spwn:node` in
automatically, since qmd declares it as a transitive dep.

Status: 🟢 working · 🟡 installed but rough · 🔴 planned.

| Dependency | Provides | Use when | Status |
|---|---|---|---|
| `spwn:unix` | bash, coreutils, grep, sed, awk, curl, jq | You need standard shell tools | 🟢 |
| `spwn:node` | Node.js 20, npm, npx | Your project uses JavaScript/TypeScript | 🟢 |
| `spwn:python` | Python 3, pip | Your project uses Python | 🟢 |
| `spwn:build` | make, gcc, g++ | You need to compile C/C++ | 🟢 |
| `spwn:git` | Git version control | You need source control (almost always) | 🟢 |
| `spwn:docker-cli` | Docker CLI (DooD) | The agent needs to manage containers | 🟢 |
| `spwn:qmd` | [QMD](https://github.com/tobi/qmd) on-device search | The agent needs to search docs, notes, or knowledge bases locally | 🟢 |
| `spwn:mempalace` | [MemPalace](https://github.com/MemPalace/mempalace) local memory palace CLI | The agent needs persistent cross-session notes | 🟡 experimental |
| `spwn:claude-code` | Claude Code CLI + pre-configured auth | You want Anthropic's agent runtime (default) | 🟢 |
| `spwn:codex` | Codex CLI installed in the image | You want OpenAI-style models | 🟡 installed, no runtime adapter wired |
| `spwn:aider` | Aider CLI | You want an open-source code-focused runtime | 🔴 planned |
| `spwn:cli` | spwn CLI inside the world | The agent needs to manage its own identity, messages, or sub-worlds | 🟢 |
| `spwn:architect` | Full orchestration daemon (includes `spwn:cli`, `spwn:claude-code`, `spwn:docker-cli`) | You're running the always-on Architect | 🟡 architect mode is in dev |

## Authoring your own

Four lines minimum:

```yaml
# spwn/tools/my-thing/tool.yaml
name: my-thing
install:
  packages: [curl]
verify:
  - command -v curl
```

Reference it from `agent.yaml#dependencies` as `tool/my-thing`.
Richer tools add `commands:` / `user-commands:` / `env:` / `files:`
plus a sibling `files/` directory referenced by `tool.yaml#files:`.
See the [internal schema](../packages/dependency/internal/manifest/schema.go)
for the full shape, or drop a new tool under
`catalog/<slug>/tools/<name>/` in the monorepo to ship a built-in.
Skills for the bundle live at `catalog/<slug>/skills/` so they stay
exposed and can be shared across every tool the entry ships.
