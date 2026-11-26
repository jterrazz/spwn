# packages/compile

The compile layer: Docker-touching image assembly.

## Role

One entry point lives here, over the Dockerfile generator and the container backend:

**`compile.New(registry, backend)` then `Builder.Build(req)`**: the base world image of one dependency set. Resolves the set into a Dockerfile, runs `docker build`, probes the result to verify every tool works. The image is tagged `<repository>:<content hash>` (`spwn/world:<hash>` at spawn time), so two dependency sets never share a tag. One build of a tag runs at a time on the machine, through a flock in the temp folder whose wait is bounded; the result holds the tag in use until its `Release`, and a build that writes a new tag removes the other tags of its repository that nothing holds and no container uses. A forced rebuild builds with no cache and never untags first. Called by `packages/architect` at spawn time.

```text
packages/transpile    →   Tree (in-memory path → bytes)
                              │
                              ▼
packages/compile      →   Docker image   (base world + derived project)
                              │
                              ▼
packages/architect    →   running container
```

Transpile is pure; compile has side effects against Docker; architect orchestrates. The split is deliberate: compile does not write agent content, does not start containers, does not parse `spwn.yaml`. It stops at "image exists."

Dependency resolution (the `Registry`, transitive expansion, topological sort) and the `CollectSkills` aggregation helper live in **`packages/dependency/resolver/`**; compile consumes them as inputs. The Docker daemon adapter lives in **`packages/container/backend/`**.

## Key types

- `Builder` / `compile.New(registry, backend)` / `Builder.Build(req)`: resolve deps, generate the Dockerfile, `docker build`, verify. The tag is the content hash, so a tag that exists is the cache.
- `BuildError`, `VerifyError`: typed errors for build failures and post-build verification failures.
- `GenerateDockerfile`, `ToolsToInputs`, `GenerateOpts`: generator seams `Builder.Build` uses.
- `base/`: embedded `world.Dockerfile` and `test.Dockerfile` templates.
- `internal/dockerfile/`: the generic Dockerfile generator fed by the tool registry.
- `internal/imagetest/`: E2E sandbox helpers for image-level tests.

## Related

- **Imported by**: `packages/architect`
- **Imports**: `packages/dependency` + `packages/dependency/resolver` (dep-resolution + aggregation helpers), `packages/container/backend` (Docker adapter), `packages/platform`
