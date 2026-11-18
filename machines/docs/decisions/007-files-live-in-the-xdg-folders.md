# ADR-007: Files live in the XDG folders

**Status:** Accepted
**Date:** 2026-10-31

## Context

Station kept everything in `~/.studio/`. The estate's rule for a CLI is binaries in `~/.local/bin`, config in `~/.config/<app>/`, state in `~/.local/state/<app>/`, caches in `~/.cache/<app>/`.

## Decision

Station follows the rule and honours each `XDG_*_HOME`. `station setup` migrates `~/.studio/` once, step by step and refusing on a conflict, and station reads only the new paths, with no fallback. The VM records and the image key, which nothing can rebuild, are named do-not-delete in the footprint page.

## Consequences

- One convention across the estate's CLIs.
- The migration is not reversible.
- Whether the rule should gain a row for data that is neither state nor cache is open, and is the estate doctrine's to decide.
