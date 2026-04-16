# packages/image

**The spwn image layer.** Owns every code path that turns a
declaration into a Docker image - both the shared base world image
(tools, skills, runtime) and the derived project images produced by
`spwn build`.

## Two entry points

There are two distinct "build an image" flows, and they live side by
side in this package because they share the tool registry, the
Docker backend, and the Dockerfile generator.

### 1. `imagebuilder.Build(req)` - the shared base world image

Called by `packages/world/architect` at spawn time. Resolves a tool
catalog into a Dockerfile, runs `docker build`, then probes the
resulting image to verify every tool actually works. Cached: if the
image already exists at the expected version label, the call is a
no-op.

This is the image every world container boots from. Generic,
multi-tool, project-agnostic.

```go
builder := image.New(registry, backend)
result, err := builder.Build(ctx, image.BuildRequest{
    BaseDockerfile: baseDF,
    Tools:          []string{"@spwn/unix", "@spwn/python"},
    Tag:            "spwn-world:latest",
    Version:        "1.0.0",
})
```

### 2. `image.BuildFromBase(ctx, cli, req)` - project-specific images

Called by the `spwn build` CLI command. Takes an existing base image
plus a compiled `Tree` and produces a derived image with the tree
baked in:

```
FROM <base>
COPY <tree>/ /world/
LABEL sh.spwn.project=<name>
LABEL sh.spwn.kind=project-build
LABEL sh.spwn.runtime=<runtime>
```

The whole build context - Dockerfile plus tree entries - is
assembled in memory and streamed into the Docker API's `ImageBuild`
endpoint. Nothing is materialised to disk.

`BuildFromBase` takes a `TreeTarballer` interface rather than a
direct `*compile.Tree`, so `packages/image` avoids a hard dependency
on `packages/compile` (which would cycle through
`packages/world → packages/image`). The concrete `*compile.Tree`
implements `Tar(io.Writer) error` and plugs in at the call site.

```go
result, err := image.BuildFromBase(ctx, dockerClient, image.BuildFromBaseRequest{
    BaseImage: "spwn-world:latest",
    Tree:      compiledTree,    // *compile.Tree satisfies TreeTarballer
    Tag:       "spwn-myproj:latest",
    Labels:    map[string]string{"sh.spwn.project": "myproj"},
})
```

## Where it sits

```
packages/compile      →   Tree (in-memory path → bytes)
                              │
                              ▼
packages/image        →   Docker image
                              │             (two flavours:
                              │              - shared base world
                              │              - project-specific)
                              ▼
packages/world        →   running container
/architect
```

Three concerns, three packages. Compile is pure. Image has side
effects against Docker. Architect orchestrates the whole pipeline
at spawn time and owns runtime state (container labels, bind
mounts, cred sync).

## Responsibilities

- **Base image selection** - pick the right starter image (the
  default world image, a custom one from `spwn.yaml`, or an
  injected `SPWN_BASE_IMAGE` for tests).
- **Tool resolution** - turn user-declared tool refs
  (`@spwn/unix`, `@spwn/python`, `@community/foo`) into concrete
  install steps via the tool catalog.
- **Layer authoring** - generate a Dockerfile that installs each
  resolved tool's `UserCommands`, adds verify hooks, and copies in
  any compile-time artifacts.
- **Derived images** - `BuildFromBase` bakes a compiled tree onto
  an existing base image, producing a pushable project artifact.
- **Build + verify** - run `docker build`, then probe the resulting
  base image to confirm every declared tool actually works.
- **Runtime config** - collect dependency `Config(runtime)` contributions
  so `architect.Spawn` can merge them into the container's runtime
  settings file after boot.

## What it does NOT do

- **It does not write agent content.** That's the compile phase.
  Agent files arrive here either as a `Tree` (BuildFromBase) or
  not at all (the shared base image knows nothing about projects).
- **It does not start containers.** That's `packages/world/architect`.
  This package stops at "image exists."
- **It does not own project parsing.** `packages/project` walks
  `spwn.yaml`; this package doesn't look at source files at all.
- **It does not manage container ↔ host syncing.** The `Backend`
  interface exposes `CopyTo` / `CopyDirTo` / `CopyDirFrom` as plumbing,
  but the *policy* (when to push, what to pull, what to allow-list) is
  owned by `packages/world/architect`.

## The `Backend` interface: why it has `CopyDirTo` / `CopyDirFrom`

`Backend` is the thin abstraction over "a running container runtime."
It has four families of methods:

- **Lifecycle**: `Create`, `Start`, `Stop`, `Remove`, `Inspect`,
  `IsRunning`, `ListContainersByLabel`.
- **Execution**: `Exec`, `ExecOutput`, `ExecDetached`.
- **Image plumbing**: `EnsureImage`, `ImageExists`, `ImageVersion`,
  `Commit`, `ImageList`, `ImageRemove`.
- **File transport**: `CopyTo` (one file), `CopyDirTo` (whole host
  directory → container, as a tar stream), `CopyDirFrom` (whole
  container directory → host, as a tar stream).

The directory transport methods exist because spwn deliberately
avoided binding `spwn/agents/<name>/` into every container. A bind
mount leaks container writes onto the host at the instant they
happen — runtime dotfiles, `.npm` caches, half-written journal
entries, per-run `CLAUDE.md` recompiles — which breaks image caching,
pollutes `git status`, and erases the isolation the container was
supposed to give. Instead:

- **`CopyDirTo` at spawn**: `filepath.Walk` over the host tree →
  stream a single tarball → `client.CopyToContainer(...)`. One
  round-trip, one logical operation, no live link.
- **`CopyDirFrom` on graceful down**: `client.CopyFromContainer(...)`
  returns a tar stream; spwn walks it, strips the top-level directory
  entry Docker adds, and writes each file to the host. Only the
  allowlisted memory dirs are snapshotted; everything else dies
  with the container.

Non-graceful termination (crash, `docker kill`) skips the sync-out
and loses any unsaved memory writes. That's an explicit trade-off:
durability is the agent's responsibility, and the allowlist is small
enough that most writes the agent actually cares about happen there.

## Relationship to `architect.Spawn`

`architect.Spawn` is the orchestrator for `spwn up`. For one spawn:

1. Validate the project, resolve workspaces, resolve the target runtime.
2. Call `imagebuilder.Build(…)` to ensure the shared base image at the
   expected version.
3. Create and start the container. The *only* host bind mounts on this
   container are the caller-declared `workspaces:` entries, mounted under
   `/workspaces/<name>/`. **No `/agents` bind, no `/world` bind from the
   project source** — the architecture moved away from both.
4. `syncAgentsInto`: for each attached agent, `CopyDirTo` the host-side
   `spwn/agents/<name>/` tree into the container at `/agents/<name>/`.
   This is a one-way snapshot at boot.
5. Write the runtime's default config files (`.claude.json`, trust
   dialogs, etc.) directly into each agent's container HOME via `CopyTo`.
6. Probe tools, then call `compile.Compile(runtime, input)` → `Tree`.
7. `materialiseWorldTree` splits the tree by prefix: `world/*` entries
   write to the host's world-state directory (mounted as `/world/` via
   a small dedicated bind), `agents/*` entries are `CopyTo`'d directly
   into the running container on top of the home tree seeded in step 4.
8. Inject runtime config, emit activity events.

On graceful `spwn down`, `syncAgentsOutOf` uses `CopyDirFrom` to pull
the four allowlisted memory directories (`journal`, `knowledge`,
`playbooks`, `skills`) back out of the container into each agent's
host-side tree. Everything else — dotfiles, npm cache, compiled
`CLAUDE.md`, `.claude/` runtime state — is discarded with the
container.

The same `compile.Tree` also powers `spwn build`, which uses
`BuildFromBase` to bake the tree into a derived image instead of
docker-cp'ing it at spawn time. Same compile phase, two different
delivery shapes: *live container injection* (spawn) vs *baked image
layer* (build).

## Testing

- Unit tests under `packages/image/...` cover registry semantics,
  config merge, and the Dockerfile generator.
- `make test-e2e-imagebuilder` runs the full build pipeline
  against real Docker, under `e2e/`.
- `tests/cli/build/build/build.e2e.test.ts` exercises
  `BuildFromBase` end-to-end via the `spwn build` CLI command,
  using the pre-built `spwn-test:latest` base image.
