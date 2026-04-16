# packages/runtimes

Built-in agent-runtime adapters.

## Role

A "runtime" is the thinking engine that actually executes an agent's prompts — Claude Code, Codex, etc. Unlike tools and skills (which live as YAML in the catalog), runtimes ship as Go code because spawn-time behavior — credential sync, default config materialisation, prelaunch shell wrapping, session resume — is too stateful for declarative YAML. Each runtime lives in its own subpackage; importing this top-level package pulls them all into the registry via `init()` side-effects.

## Key types

- `All []image.Tool` — the list of every built-in runtime, exposed as image Tools so the build pipeline can include them like any other dependency.
- `RegisterDefaults(*image.Registry)` — register every built-in runtime into a registry.
- `claude_code/` — the Claude Code runtime (image Tool + compile renderer in `compile/` + spawn adapter in `adapter/`).
- `codex/` — the Codex runtime (image Tool only today; no spawn adapter wired).

## Related

- **Imported by** — `apps/cli` (to register runtimes into the catalog), `catalog` (for dep-resolution tests), `packages/architect` (to build runtime images), `packages/image`, `packages/world`
- **Imports** — `packages/dependency`, `packages/image`, `packages/world/runtime` (the Runtime port interface)
