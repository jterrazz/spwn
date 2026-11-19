# ADR-017: One repository, station moves in under machines/

**Status:** Proposed
**Date:** 2026-11-18

An agent leaves the status `Proposed`; only the owner writes `Accepted`.

## Context

Two products serve one owner's day: Spwn, which runs agents in worlds, and station (`jterrazz/studio-station`), which sets up and runs a Mac, its VMs and his other Macs. They are released, installed and documented apart, and the owner wants them in one repository with one release and one installer. The longer plan moves responsibilities from the agents side to the machines side, which is easier to do inside one tree than across two.

Both repositories are public and MIT. Neither has real users to keep compatible: the first unified release is an alpha that may half-work.

Station is under active development by its own sessions, on branches and worktrees of its own. Whatever moves must let them carry that work across.

## Decision

Station's whole tree moves into this repository under one prefix, `machines/`, with its history, its internal layout intact and nothing renamed.

- **History is kept.** The import is a `git subtree add` without `--squash`: station's commits keep their shas, so a branch cut from station's `main` before the import is replayed onto this repository's `main` by `git rebase -X subtree=machines`, or by `git cherry-pick -X subtree=machines` commit by commit.
- **The import happens at a freeze.** Station's orchestrator names the freeze sha on station's `origin/main`; this record's rehearsal took `58906e8b7` on 2026-11-18. Station's tags are not imported: its `v1.5.0` to `v3.1.0` would collide with Spwn's `v1.x`.
- **Nothing is renamed in the import.** The Go module path `github.com/jterrazz/studio-station/...`, the binary `station`, the bundle ids, sockets, state folders and goldens all stay. Each rename is a later change of its own.
- **The two trees cohabit with the least glue.** No `go.work`, package manager or `apps/` folder is merged. The root declares `machines` a verbatim root in `studio.config.ts`, so no root gate reads it; `make machines-check` and `make machines-build` delegate to station's own `Makefile`; `check.yaml` gains one job that runs the shared check action inside `machines/`; station's nested workflows, dead once nested, are removed.
- **One tag releases both.** `publish.yaml` builds station's linux and darwin binaries and `Station.app.zip`, then GoReleaser opens the one release with Spwn's archives and station's files as extra files, listed in the one `checksums.txt`.
- **The installed names are the long ones.** The installer puts `spwn` and `station` in `~/.local/bin`, adds `spwn-agents` and `spwn-machines` beside them as links, and writes the shell aliases `agents` and `machines` to the profile. The built binaries keep their names until the rename lands.

## Consequences

- One release number serves two products, and the line continues after Spwn's last release, `v0.20.0`: the first unified tag is `v0.21.0-alpha.1`. Station's last release was `v3.1.0` in its former repository and its minimum accepted version is `v0.1.0`, so the tag passes; the two installed station hosts are reinstalled by hand. The alpha is published as a normal release marked latest, not a pre-release, so `releases/latest` (station's installer, its menubar script and its release fetcher) returns it; the word `alpha` carries the state.
- Station's code still names its old repository where it fetches its own release (`apps/engine/images/calls/binary.go`), and refuses a tag with a pre-release label there; until that changes, an image built from an alpha needs its linux binary given by hand.
- Station keeps its own Go version (1.27, from `machines/go.work`) beside Spwn's (1.25); CI provisions each from its own file.
- The GoReleaser changelog of the first unified tag reads station's whole imported history as new commits.
- Station's manual keeps describing its former workflows and installer until its sessions update it; its links within `machines/` keep resolving.

### Later

- **The binaries are renamed** `spwn-machines` (station) and `spwn-agents` (spwn); the aliases `machines` and `agents` stay the short words.
- **Whether a `spwn` umbrella command exists** above the two is open: the owner will rework the names, and nothing here decides it.
- **Responsibilities move** from the agents side to the machines side, one by one.
- **The module path, bundle ids, sockets and state folders** follow the rename, each as its own change.
- **The GUI app** is named Spwn too.
