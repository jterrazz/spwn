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
- **Plugin config** - collect plugin `Config(runtime)` contributions
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

## Relationship to `architect.Spawn`

`architect.Spawn` is the orchestrator for `spwn up`. For one spawn:

1. Validate the project and resolve the target runtime.
2. Call `compile.Compile(runtime, input)` → `Tree`.
3. Call `imagebuilder.Build(…)` to ensure the shared base image.
4. Create and start the container, bind-mounting the materialised
   `Tree` into the container's `/world` path.
5. Inject plugin runtime config, probe tools, emit activity events.

Steps 2 and 3 are separate from step 3 in `spwn build`, which uses
`BuildFromBase` to bake the `Tree` directly into a derived image
instead of bind-mounting it at runtime. Same compile phase, two
different delivery shapes.

## Testing

- Unit tests under `packages/image/...` cover registry semantics,
  config merge, and the Dockerfile generator.
- `make test-e2e-imagebuilder` runs the full build pipeline
  against real Docker, under `e2e/`.
- `tests/cli/build/build/build.e2e.test.ts` exercises
  `BuildFromBase` end-to-end via the `spwn build` CLI command,
  using the pre-built `spwn-test:latest` base image.
