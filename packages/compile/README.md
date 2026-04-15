# packages/compile

**The spwn compiler.** Translates a provider-neutral spwn project
(`spwn.yaml` + `spwn/agents/*` + skills + hooks) into a
runtime-specific file layout that a concrete agent runtime — Claude
Code today, Codex tomorrow — can boot from.

## The pitch in one line

Spwn is a compiler the same way `tsc` is a compiler: you author in a
portable source language, and the compiler emits files your target
runtime understands. You never write a `.js` file by hand for TypeScript;
you should never write a `CLAUDE.md` by hand for spwn.

## The two-phase model

```
                  spwn source                 target runtime
                  ───────────                  ──────────────
                 ┌──────────┐                 ┌──────────┐
spwn.yaml  ───►  │          │  ──► Tree ──►  │  Docker  │
AGENT.md   ───►  │ compile  │    (files)     │  image   │
skills/*   ───►  │ (this    │                │ (image   │
hooks/*    ───►  │ package) │                │ package) │
                 └──────────┘                 └──────────┘
                     pure                    side-effectful
```

Phase 1 — **compile** (this package) — is a pure function. Given an
`Input` it returns a `*Tree`, a sorted, in-memory `path → bytes`
map. No disk writes, no Docker calls. The same input produces the
same output.

Phase 2 — **link** (`packages/image`) — consumes the `Tree` and
turns it into a Docker image. This is where the side effects live:
`docker build`, tool verification, runtime state.

Analogy:

| tsc                 | spwn                          |
| ------------------- | ----------------------------- |
| `.ts` source files  | `spwn.yaml` + `spwn/**`       |
| typed AST + checker | `compile.Input` + validator   |
| JS codegen          | `Runtime.Render`              |
| `outDir` of `.js`   | `Tree` / materialised layout  |

## Source format (committed, provider-neutral)

A spwn project looks like this:

```
my-repo/
├── spwn.yaml
└── spwn/
    ├── agents/
    │   └── neo/
    │       ├── agent.yaml              # composition: tools, skills, profile
    │       ├── AGENT.md                # the agent's prompt (provider-neutral)
    │       ├── core/profile.md         # identity
    │       ├── skills/                 # procedures and checklists
    │       ├── knowledge/              # learned facts
    │       ├── playbooks/              # promoted workflows
    │       └── journal/                # session history
    ├── skills/                         # shared skills across agents
    └── hooks/                          # lifecycle hooks (build-time, spawn-time)
```

None of these files know what runtime will execute them. The
per-agent prompt is `AGENT.md`, not `CLAUDE.md`, and nothing under
`spwn/` references specific runtime conventions by name. That
discipline is what makes one project portable across every backend
the compiler grows to support.

## Target runtimes

Today: **Claude Code**, under `packages/compile/runtimes/claudecode`.
It knows that Claude Code reads `CLAUDE.md` from the working
directory on boot and that the `/world/` mount expects files named
`physics.md`, `faculties.md`, `AGENTS.md`, and `roster.md`.

Tomorrow: Codex, Open Interpreter, custom local runners. Each will
be a new sub-package under `runtimes/` that implements the
`compile.Runtime` interface. Adding a runtime **does not** touch the
source format.

## The `Tree` type

`Tree` is intentionally simple: a flat `map[string][]byte` with
ergonomic helpers.

```go
tree := compile.New()
tree.AddString("world/physics.md", physics)
tree.AddString("agents/neo/CLAUDE.md", entrypoint)

for _, p := range tree.Paths() {           // sorted
    data, _ := tree.Get(p)
    // write, hash, diff, inspect...
}

_ = tree.WriteTo("/tmp/out")                // materialise
```

Why a flat map:

- **Deterministic** — `Paths()` is sorted, so golden fixtures
  diff cleanly.
- **Composable** — `WriteTo` puts the tree anywhere. `Walk` lets
  callers route different prefixes to different destinations, which
  is what `architect.Spawn` does today (it splits `world/*` and
  `agents/*` into two bind-mounted host directories).
- **Testable** — comparing two trees is a map comparison. No temp
  dirs, no file system assertions.

When a renderer eventually needs streaming (multi-gigabyte tool
bundles, for example), the `Tree` interface can grow without
breaking callers — `Add` becomes `AddReader`, `Walk` becomes a
channel. That's a forward refactor; today the flat map is enough.

## The `Runtime` interface

```go
type Runtime interface {
    Name() string
    Render(input Input) (*Tree, error)
}
```

A `Runtime` is a **pure function from `Input` to `Tree`**. No I/O,
no Docker, no time-dependent output. Three reasons this matters:

1. **Caching is a map lookup.** Same input, same tree, same image.
2. **Tests are one-liners.** Call `Render`, compare paths, compare
   contents. Golden fixtures (coming in Phase 4) diff two maps.
3. **Reasoning.** When a bug appears in the rendered output, you
   know it's the input or the renderer — never a race, never a
   dangling file from a previous run.

### Adding a new runtime

1. Create `packages/compile/runtimes/<name>/` with a Go file
   declaring `package <name>`.
2. Implement a `Runtime` struct whose `Name()` returns the runtime
   identifier (e.g. `"codex"`) and whose `Render` produces a `Tree`.
3. Register via `init()`:

   ```go
   func init() { compile.Register(&Runtime{}) }
   ```

4. Callers reach the runtime via `compile.Compile("<name>", input)`.

## `Input` and growth

`Input` is the struct the compiler hands every `Runtime`:

```go
type Input struct {
    Manifest      models.Manifest
    VerifiedTools []string
    WorldID       string
    Agents        []AgentInput
}
```

It exists so future fields (profiles, hooks, per-agent overrides,
target platform) can land without breaking every runtime. The
design rule is: a `Runtime` only reads from `Input`; it never
touches the disk, the network, or global state.

Phase 1 hand-constructs `Input` inside `architect.Spawn` from the
data it already has. Phase 2+ will add a loader that reads
`Input` from a project directory so `spwn compile ./my-project` can
run without spinning up a live world.

## Relationship to the CLI

| Command      | What it does                                        |
| ------------ | --------------------------------------------------- |
| `spwn check` | Parses + validates the project. (Phase 3 will run a full dry-run compile through this package.) |
| `spwn build` | Runs compile + link, produces a Docker image. (Phase 2 will route it through `compile.Compile`.) |
| `spwn up`    | Runs compile + link + container boot.               |

Today `spwn up` is the only path that exercises the compiler in
production (via `architect.Spawn` → `compile.Compile`). The CLI
commands that expose the compiler as a first-class verb land in
Phase 2.

## Testing strategy

Phase 1 ships unit tests for the `Tree` type (CRUD, sorting,
`WriteTo` round-trip) and the existing `claudecode` generator tests
that moved over from the old `worldfiles` package.

Phase 4 will add **golden-fixture tests**: a small spwn project
checked into `testdata/`, a recorded expected `Tree` (map of
path → content hash), and a diff on every CI run. When a renderer
changes output by even one byte, the test fails with a readable
diff. That's the payoff of making `Render` pure.

## Why provider neutrality matters

- **Portability.** The same project runs on every backend the
  compiler supports. Switching runtimes is a flag change, not a
  source rewrite.
- **Testability.** The neutral IR (`Input → Tree`) is easy to fake,
  easy to snapshot, and never races with real containers.
- **No vendor lock-in.** If a runtime disappears or changes its
  file conventions tomorrow, you delete one sub-package. Your repo
  doesn't move.
- **Reviewability.** A diff in `spwn/agents/neo/AGENT.md` means
  "the agent's behavior changed." A diff in `CLAUDE.md` inside a
  generated artifact would conflate agent behavior and compiler
  output.

If spwn had started with `CLAUDE.md` in the source tree (it did,
briefly), it would already be harder to add a second runtime. This
package is the architecture that prevents that from ever being true
again.
