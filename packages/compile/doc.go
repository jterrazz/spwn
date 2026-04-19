// Package compile owns every Docker-touching concern: Dockerfile
// generation, Docker backend adapter, image build pipeline,
// post-build tool verification.
//
// Two entry points share the registry, backend, and generator:
//
//   - Builder.Build (constructed via compile.New) — the shared base
//     world image. Resolves a dependency set into a Dockerfile, runs
//     docker build, probes the result. Cached on a version label.
//   - BuildFromBase — project-specific derived images. Composes a
//     transpiled Tree onto a base image via a single streamed tar
//     build context.
//
// Sub-packages:
//
//   - base/    — embedded world.Dockerfile / architect.Dockerfile.
//   - e2e/     — Docker-backed integration tests (build tag: e2e).
//   - internal/dockerfile/ — Dockerfile generator internals.
//   - internal/imagetest/  — integration-test harness.
//
// The package does not parse spwn.yaml (that's packages/dependency),
// does not write agent content (that's packages/transpile), and does
// not start containers (that's packages/architect). It stops at
// "image exists."
package compile
