# packages/image

**The spwn linker.** Takes a compiled `Tree` (from
`packages/compile`) plus a resolved tool list and bakes them into a
Docker image that a world container can boot from.

If `packages/compile` is `tsc`, this package is the bundler: it
takes the emitted artifacts, adds the runtime dependencies, and
produces the single distributable output.

## Where it sits

```
packages/compile   →   Tree (in-memory path → bytes)
                          │
                          ▼
packages/image     →   Docker image  (layered, content-addressed)
                          │
                          ▼
packages/world     →   running container
/architect
```

Three concerns, three packages. Compile is pure; image has side
effects against Docker; architect orchestrates the whole pipeline
and owns runtime state (container labels, bind mounts, cred sync).

## Responsibilities

- **Base image selection** — pick the right starter image (spwn's
  default `WorldImage`, a custom one from `spwn.yaml`, or an
  injected `SPWN_BASE_IMAGE` for tests).
- **Tool resolution** — turn user-declared tool refs
  (`@spwn/unix`, `@spwn/python`, `@community/foo`) into concrete
  install steps via the tool catalog.
- **Layer authoring** — generate a Dockerfile that installs each
  resolved tool's `UserCommands`, adds verify hooks, and copies in
  any compile-time artifacts.
- **Build + verify** — run `docker build`, then probe the resulting
  image to confirm every declared tool actually works.
- **Plugin config** — collect plugin `Config(runtime)` contributions
  so `architect.Spawn` can merge them into the container's runtime
  settings file after boot.

## What it does NOT do

- **It does not write agent content.** That's the compile phase.
  Agent files arrive here as a `Tree` the caller already built.
- **It does not start containers.** That's `packages/world/architect`.
  This package stops at "image exists, image is verified."
- **It does not own project parsing.** `packages/project` walks
  `spwn.yaml`; this package doesn't look at source files at all.

## Relationship to `architect.Spawn`

`architect.Spawn` is the orchestrator. For one `spwn up`:

1. Validate the project and resolve the target runtime.
2. Call `compile.Compile(runtime, input)` → `Tree`.
3. Call `image.Build(…)` to produce the Docker image.
4. Create and start the container, bind-mounting the materialised
   `Tree` (via `Tree.WriteTo` into host state directories).
5. Inject plugin runtime config, probe tools, emit activity events.

Steps 2 and 3 are the two compiler phases. This package owns step
3 and the tool catalog resolution that feeds it.

## Testing

- Unit tests under `packages/image/...` cover registry semantics,
  config merge, and the Dockerfile generator.
- `make test-e2e-imagebuilder` runs the full build pipeline
  against real Docker, under `e2e/`.

## Future: `spwn build`

Phase 2 of the compiler refactor wires a top-level `spwn build`
CLI command that goes compile → image → named artifact, without
spawning a container. This package is already shaped for that use
case — `Build` returns the image tag and a build manifest — the
CLI just needs a thin wrapper.
