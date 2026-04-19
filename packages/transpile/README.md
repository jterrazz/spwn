# packages/transpile

The spwn transpiler — provider-neutral source → runtime-specific file tree.

## Role

Spwn is a transpiler the same way `tsc` is: you author a portable source language (`spwn.yaml` + `spwn/agents/*` + skills + hooks), and the transpiler emits files a concrete runtime (Claude Code today, Codex tomorrow) understands. You never write a `.js` file by hand for TypeScript; you should never write a `CLAUDE.md` by hand for spwn.

The transpile phase is a **pure function**: `Input → *Tree`. No disk writes, no Docker. A `Tree` is a sorted, in-memory `path → bytes` map — same input, same bytes, deterministic for golden tests. Materialisation (writing the tree into a container or onto disk) is the compile phase, owned by `packages/compile`. Runtime-specific renderers live in `packages/runtimes/<runtime>/render.go` — thin layout adapters that read content from `worldbook/` (spwn's opinionated world defaults) and place files at runtime-specific paths.

```
        spwn source                   target runtime
        ───────────                    ──────────────
       ┌───────────┐                  ┌──────────┐
spwn.yaml ─►│           │ ──► Tree ──►│  Docker  │
AGENTS.md ─►│ transpile │   (files)   │  image/  │
skills/*  ─►│ (this     │             │ container│
hooks/*   ─►│ package)  │             │          │
       └───────────┘                  └──────────┘
           pure                      side-effectful
```

Powers two delivery shapes, sharing the transpile phase verbatim:

1. **Spawn-time injection** (`spwn up`) — the tree is docker-cp'd into a running base container. Every spawn re-transpiles; no rebuild needed for source edits.
2. **Build-time compile** (`spwn build`) — the same tree is `COPY`'d into a derived image at `docker build` time. Produces a self-contained artifact you can push and run without the source tree.

## Key types

- `Input` — the source snapshot handed to every renderer: manifest, verified tools, selected world, agents with their layers, imports, hooks.
- `Tree` — flat `path → bytes` map. `AddString` / `AddBytes`, sorted iteration, `WriteTo(dir)` for host-side materialisation.
- `Runtime` interface — `Name()` + `Render(Input) → *Tree`. Pure. Implementations live in `packages/runtimes/<name>/`.
- `Compile(name, input) → *Tree` — look up the registered runtime and render.
- `source/` sub-package — `Load(root)` walks a project directory into a `ProjectSource`; `ToCompileInput(source, worldName)` projects it onto an `Input`.
- `worldbook/` sub-package — spwn's opinionated world content (physics, faculties, roster, architect identity + skills). Runtime-neutral; imported by every runtime renderer and by `packages/architect` for image-build + hot-deploy.

## Related

- **Imported by** — `apps/cli` (`spwn build`, `spwn check --deep`), `packages/architect` (spawn pipeline), `packages/runtimes/*` (renderers)
- **Imports** — `packages/project`, `packages/agent`, `packages/dependency`
