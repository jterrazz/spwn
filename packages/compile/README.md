# packages/compile

The compile layer — Docker-touching image assembly.

## Role

Two entry points live here because they share the Dockerfile generator and the container backend:

1. **`compile.New(registry, backend) → Builder.Build(req)`** — the shared base world image. Resolves a dependency set into a Dockerfile, runs `docker build`, probes the result to verify every tool works. Cached on a content-addressed version label. Called by `packages/architect` at spawn time.
2. **`compile.BuildFromBase(ctx, cli, req)`** — project-specific derived images. Takes a base image plus a transpiled `Tree` (from `packages/transpile`) and produces `FROM <base> / COPY tree/ /world/` as a pushable artifact. Called by `spwn build`.

```
packages/transpile    →   Tree (in-memory path → bytes)
                              │
                              ▼
packages/compile      →   Docker image   (base world + derived project)
                              │
                              ▼
packages/architect    →   running container
```

Transpile is pure; compile has side effects against Docker; architect orchestrates. The split is deliberate: compile does not write agent content, does not start containers, does not parse `spwn.yaml`. It stops at "image exists."

Dependency resolution (the `Registry`, transitive expansion, topological sort) and auxiliary aggregation helpers (`CollectSkills`, `CollectRuntimeConfigs`, `MergeRuntimeConfig`) live in **`packages/dependency/resolver/`** — compile consumes them as inputs. The Docker daemon adapter lives in **`packages/container/backend/`**.

## Key types

- `Builder` / `compile.New(registry, backend)` / `Builder.Build(req)` — resolve deps → generate Dockerfile → `docker build` → verify. Result cached on content-addressed version label.
- `BuildFromBase(ctx, cli, req)` — compose a `TreeTarballer` onto a base image. Interface (not concrete `*transpile.Tree`) to avoid a `compile → transpile → compile` cycle.
- `BuildError`, `VerifyError` — typed errors for build failures and post-build verification failures.
- `GenerateDockerfile`, `ToolsToInputs`, `GenerateOpts` — generator seams used by both entry points.
- `base/` — embedded `world.Dockerfile`, `architect.Dockerfile`, `test.Dockerfile` templates plus `entrypoint.sh`.
- `internal/dockerfile/` — the generic Dockerfile generator fed by the tool registry.
- `internal/imagetest/` — E2E sandbox helpers for image-level tests.

## Related

- **Imported by** — `apps/cli` (`spwn build`), `packages/architect`
- **Imports** — `packages/dependency` + `packages/dependency/resolver` (dep-resolution + aggregation helpers), `packages/container/backend` (Docker adapter), `packages/platform`
