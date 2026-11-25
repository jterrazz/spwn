# ADR-004: station status --json is a contract

**Status:** Superseded: the hidden `station status` and `station.status/v1` were deleted with v2's other aliases; `studio doctor` reads `station overview --json`, `station.overview.read/v1`
**Date:** 2026-10-31

## Context

`studio doctor` re-probed binaries and the tailnet itself, duplicating what station already knows about what is installed.

## Decision

`station status --json` prints `station.status/v1`, a stable document, and `studio doctor` reads it. A field is only added within v1; a rename or removal makes `v2`. The Go structs behind it are the only definition, pinned by a test that encodes a fixed snapshot.

## Consequences

- Station owns what is installed on a machine; doctor owns judging it.
- The contract constrains refactors of the status code.
- Doctor reports station absent or unreadable as one line instead of guessing.
