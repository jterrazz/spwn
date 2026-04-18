# packages/image

The image layer — every Docker-touching concern, from Dockerfile generation to container lifecycle.

## Role

Two entry points live here because they share the tool registry, the Docker backend adapter, and the Dockerfile generator:

1. **`imagebuilder.Build(req)`** — the shared base world image. Resolves a dependency catalog into a Dockerfile, runs `docker build`, probes the result to verify every tool works. Cached on a version label. Called by `packages/architect` at spawn time.
2. **`image.BuildFromBase(ctx, cli, req)`** — project-specific derived images. Takes a base image plus a transpiled `Tree` (from `packages/compile`, historical name) and produces `FROM <base> / COPY tree/ /world/` as a pushable artifact. Called by `spwn build`.

```
packages/compile      →   Tree (in-memory path → bytes)
                              │
                              ▼
packages/image        →   Docker image   (base world + derived project)
                              │
                              ▼
packages/architect    →   running container
```

Transpile (packages/compile — historical name) is pure; `image` has side effects against Docker; architect orchestrates. The split is deliberate: `image` does not write agent content, does not start containers, does not parse `spwn.yaml`. It stops at "image exists."

## Key types

- `imagebuilder.Build(req)` / `image.New(registry, backend)` — resolve deps → Dockerfile → docker build → verify. Result cached on version label.
- `image.BuildFromBase(ctx, cli, req)` — compile a `TreeTarballer` onto a base image. Interface (not concrete `*compile.Tree`) to avoid a `image → compile → image` cycle.
- `Backend` (in `backend/`) — thin abstraction over "a running container runtime". Four families: lifecycle (`Create`/`Start`/`Stop`), execution (`Exec`), image plumbing (`EnsureImage`/`Commit`), file transport (`CopyTo` / `CopyDirTo` / `CopyDirFrom`). `CopyDirTo`+`CopyDirFrom` exist because spwn deliberately avoided binding `spwn/agents/<name>/` — tar-stream snapshots at boot/shutdown preserve container isolation without leaking runtime dotfiles onto the host.
- `Registry` / `Tool` — the in-memory dependency catalog; tools are registered here and resolved transitively before Dockerfile generation.
- `base/` — embedded `world.Dockerfile`, `architect.Dockerfile`, `test.Dockerfile` templates plus `entrypoint.sh`.
- `backend/` — the Docker client adapter (the only concrete `Backend` today).
- `internal/dockerfile/` — the generic Dockerfile generator fed by the tool registry.
- `probe/` — post-build verification (each tool's `verify:` commands run inside the image).

## Related

- **Imported by** — `apps/api`, `apps/cli`, `catalog`, `packages/architect`, `packages/runtimes`, `packages/world`
- **Imports** — `packages/dependency` (for parsing tool manifests via the adapter), `packages/platform`
