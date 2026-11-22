# ADR-075: A move merges into a cache and refuses onto anything else

**Status:** Proposed
**Date:** 2026-11-22

## Context

`station disk move` was documented as refusing any destination that holds anything. For `~/.npm` that made the offered fix dead: npm writes `_logs` and its update check under its cache on every call, so once `~/.cache/npm` held a `_logs` of its own, every later move of a `~/.npm` that came back failed with `move ~/.cache/npm/_logs: file already exists` and a generic hint. The two copies are the same rebuildable cache; a refusal there protects nothing. A destination that holds data (`~/.local/share/go/bin`) or a whole folder's destination (`~/.local/share/station/vms`) is another matter: what is there may be the owner's only copy.

## Decision

A move is planned whole before anything moves. An entry of a misplaced folder whose name its app's cache destination (a `cache`-level path of its row) already holds is dropped from the misplaced folder, and the destination's copy is kept. Any other destination that holds something refuses the move with `disk.destination-held`, naming the path and what to do (`move or remove <path>, then station disk move <folder>`), and nothing moves.

A move also writes its app's relocation, as `station init` writes it, so the app looks at the new place and the folder does not come back.

## Consequences

- A cache's older copy of an entry is lost on a move; it is rebuildable by definition of its level.
- A data or whole-folder collision still needs the owner, now told by path instead of by an error of the filesystem.
- A move of `~/go` now also writes Go's env file and mise's `go_set_gobin`, as init does.
