# ADR-046: The repository is a monorepo of members

**Status:** Proposed
**Date:** 2026-11-13

## Context

The repository grew from one Go module with an engine at `src/`, a Swift app at `apps/menubar`, a catalog embedded from inside the engine's package, the specs at `specs/`, and a root that declares a `cli` kind. The engine's folder is called nothing but `src`, the data the engine embeds lives under the code that reads it, and the next members (the window, the file-format schemas, the API's package) have no place that says what they are. [ADR-001](001-station-mirrors-the-studio-cli-layers.md) and [ADR-014](014-families-and-contracts.md) cut the code into layers and families inside one tree; neither says what a member of the repository is. The estate's other repository that spans Go and Node, spwn, has a shape for it.

## Decision

- **The root declares `workspaces` and no kind.** Each member is a folder the root lists: `apps/engine` (Go), `apps/menubar` (Swift), `apps/window` (the window, when it lands), `packages/api` (the schemas and API package) and `specs`.
- **`catalog/` is the one data folder, and its own Go module.** It holds `packages/<category>` and `machines`, and exposes them as an `embed.FS` the engine imports, so any checkout builds with no generate step. The alternatives lost: a gitignored mirror of the folder into the engine (every build, test and release would need a generate first, and a fresh clone would not build), and the data kept under the engine (the owner decided one catalog folder, and a later extraction to a store of its own).
- **The engine's Go module path is `github.com/jterrazz/studio-station/apps/engine`**, the catalog's `.../catalog`, the specs' `.../specs`. Each module requires its siblings and carries a `replace` to their folder, so a build works with or without `go.work`.
- **The toolchain is npm workspaces, `go.work` and Turborepo.** Turborepo carries what is per-module and cacheable: Go `lint` and `test` per module and the specs' test. Version-stamped builds, `generate` and the passes whose result depends on the machine stay direct Makefile lines, since a version read from `git describe` is not a content hash.
- **A Go member has no `package.json`.** Only a node member (the specs, the window, the API package) carries one and a `studio.config.ts`.
- **A move to pnpm is the owner's.** spwn runs pnpm workspaces under Turborepo; the stack doctrine here says `npm` only, never pnpm (`docs/concepts/stack.md`). This record keeps npm until the owner changes that doctrine in its own decision, so no page says what the repository does not.

## Consequences

- A lot moves the engine to `apps/engine`, the data to `catalog`, and rewrites the paths the architecture specs, the build and the release read; scripted, one commit per script.
- The release builds `./apps/engine/cmd/station` with the same version flag, under the module's new path.
- Branches held on the old layout are replayed through the same scripts, not merged by hand.
- If the owner chooses pnpm, the change is `workspaces` in a `pnpm-workspace.yaml`, the lock file and the install command, and a doctrine commit that lands first.
- The engine's catalog row of the architecture rules reads the embedded files through the catalog module, a swap of `embed` for the module's path that keeps the row at three entries.
- A Go member without `studio.config.ts` gives the studio toolchain nothing to judge; the Go modules are judged by their own lint and tests.
