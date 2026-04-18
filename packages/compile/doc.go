// Package image owns every Docker-touching concern: Dockerfile
// generation, Docker backend adapter, image build pipeline,
// post-build tool verification.
//
// Two entry points share the registry, backend, and generator:
//
//   - imagebuilder.Build — the shared base world compile. Resolves a
//     dependency catalog into a Dockerfile, runs docker build,
//     probes the result. Cached on a version label.
//   - BuildFromBase — project-specific derived images. Compiles a
//     transpile.Tree (currently named transpile.Tree for historical
//     reasons) onto a base image via a single streamed tar build
//     context.
//
// Sub-packages:
//
//   - backend/ — Docker client adapter (the only concrete Backend).
//   - base/    — embedded world.Dockerfile / architect.Dockerfile.
//   - probe/   — post-build verification (tool Verify commands).
//   - internal/dockerfile/ — generator internals.
//   - internal/imagetest/  — integration-test harness.
//
// The package does not parse spwn.yaml (that's packages/dependency),
// does not write agent content (that's packages/compile), and does
// not start containers (that's packages/architect). It stops at
// "image exists."
package compile
