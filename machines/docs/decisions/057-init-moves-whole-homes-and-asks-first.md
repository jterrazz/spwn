# ADR-057: Init moves an app's whole old home, and says so and asks before it moves

**Status:** Proposed
**Date:** 2026-11-18

## Context

`station init` writes each app's own settings file so the app keeps its files in the XDG folders
([Disk](../reference/disk.md#relocation)), then moves what the app already left at the home root.
On a Mac mini with a used `~/.bun` and `~/go` (2026-11-18), init moved bun's globals and bins and
Go's bins, and left `~/.bun/install/cache`, `~/.bun/_bun`, `~/go/pkg/sumdb` and the whole `~/.npm`
behind: the first `station disk` after init flagged `~/.bun`, `~/go` and `~/.npm` as misplaced,
three folders station itself had just relocated. It also moved 232 MB, and on a Mac with VMs
`~/.tart`'s tens of GB, without a word before it did.

On 2026-11-18 the owner ruled: the relocation of third-party files is kept, and it is complete and
announced before it moves anything.

## Decision

- **A relocation moves the whole old home.** Each part an app reads lands where its settings now
  point: bun's cache under `install.cache.dir`, Go's module cache under `GOMODCACHE` and its
  checksum database's state under `GOPATH/pkg/sumdb`, every entry of `~/.npm` under npm's `cache`.
  A file the app writes again on its own is deleted instead (`~/.bun/_bun`, bun's zsh completion).
  An entry whose destination already holds one of its name stays, and init says so.
- **Init says what it moves before it moves anything:** `# To move`, one row per move, from where,
  to where, and its size, or the file it deletes.
- **On a terminal it asks `Move them?`.** A no ends init there, exit 1: no file moves and no step
  runs, since an app's settings written without its files moved would point it at an empty folder.
- **Off a terminal it is refused, exit 2, before any step**, and `--yes` answers the question, as
  a Mac with no name yet is refused there and `--name` names it.
- **With nothing to move, init prints nothing more and asks nothing.**

The `setup.init` call carries the question as every confirmed call does (`Confirm: relocate`, its
`Confirm` input answered `yes`), and its `Plan` input returns what it would move with nothing done.

## Consequences

- A script that runs `station init` on a Mac whose apps left files at the home root passes
  `--yes`; `install.sh` passes its own arguments on.
- `station disk` right after init flags none of the folders init relocated.
- `station disk move` and init read the same catalogue rows, so a folder moves the same way
  whichever runs.
