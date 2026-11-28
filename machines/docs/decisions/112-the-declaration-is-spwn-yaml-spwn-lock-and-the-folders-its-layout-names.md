# ADR-112: The declaration is spwn.yaml, spwn.lock and the folders its layout names

**Status:** Proposed
**Date:** 2026-11-28
**Amends:** [ADR-099](099-project-apply-writes-only-what-its-plan-lists-and-removes-only-what-it-wrote.md): its decision 6, whose declaration was everything under `spwn/`.

An agent leaves the status `Proposed`; only the owner writes `Accepted`.

Follows root [ADR-024](../../../docs/decisions/024-a-projects-folders-sit-at-its-root.md): a project's folders sit at its root, renamed one at a time by `folders:`, and the old `spwn/` layout is refused.

## Context

ADR-099 §6 named the declaration `spwn.yaml`, `spwn.lock` and everything under `spwn/`, and walked it one name at a time before any of it was read. With the folders at the root, "everything under `spwn/`" would become everything at the root: the repository's own code, its `.git`, whatever a clone holds.

## Decision

1. **The declaration is three parts.** `spwn.yaml` and `spwn.lock`, walked as files, then each folder the project's layout names (`agents/`, `skills/`, `tools/`, `hooks/`, `commands/`, `profiles/`, or what `folders:` renames them to). Nothing else at the root is walked, listed or read; a link or a FIFO elsewhere at the root is not the declaration's.
2. **The layout is read before the folders.** Between the two files and the folders, the layout is read from `spwn.yaml` through the root opened at the project (`spwnproject.LayoutAt`): a project of the old layout is refused as `project.old-layout`, and a `folders:` entry that names no folder of the project as `project.folders-invalid` naming its key, both before any folder is walked. The refusal is the root module's sentence, so both binaries say the same.
3. **The rest of §6 holds, with the new names.** Every read of the declaration goes through the root opened at the project; a tool's folder is read through a root opened at its own folder of the layout's tools folder, and that folder, or the tools folder itself, that is a link is refused before the root is opened; "an agent that is not one folder name of `spwn/agents/`" reads "of the layout's agents folder".
4. **A fake world tells the same layout.** The fake projects take the real adapter's reading of a layout from a project's files (`manifest.Folders`, over the root module's `LayoutOfFiles`, by kind), handed in by the wiring, since no fake imports an Agents package.
5. **The walk has one budget.** The folders of a layout hold at most 4,096 names together, a folder counted with its files, none deeper than 16 names from the root, and no file larger than 4 MiB (`project.Budget`); the first name past it is refused as `project.too-large`, before any of the declaration is read. A repository is a project at its root, so a folder of one of the six names may hold the repository's own tree.
6. **A refusal in a layout's folder says how to move it.** A link, a name neither a file nor a folder, or a name past the budget under a kind's folder is refused naming its path and the `folders:` line that moves that kind (`folders: skills: <folder>`); the root module's loaders say the same of an `agent.yaml` or a hook that does not parse.

## Consequences

- A world's knowledge folder is not walked. No host code reads inside it: it is checked to be a folder (by `os.Stat` in `packages/architect/spawn.go`, through the project's root in `packages/transpile/source/input.go` and in the knowledge rule of `packages/project/internal/validate`) and bound into the world at `/world/knowledge`, where only the world reads it.
- A repository whose root already uses one of the six names for something else, a `hooks/` of git hooks or a `tools/` of scripts, declares `folders:` to move that kind elsewhere; until it does, that folder is walked and read as the kind's.
- The project-plan stories `walk-only-declared-folders`, `project-old-layout`, `project-folders-invalid` and `project-too-large` (with those of apply, drift and show) prove the boundary over the fake world.
