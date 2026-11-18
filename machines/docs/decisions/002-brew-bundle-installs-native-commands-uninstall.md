# ADR-002: brew bundle installs, native commands uninstall

**Status:** Accepted
**Date:** 2026-10-31

## Context

Install and upgrade were hand-dispatched per package kind, twice, and disagreed. `brew bundle` covers brew, cask, mas, npm and uv and installs and upgrades by default. It cannot uninstall one entry: `remove` only edits a Brewfile and `cleanup` removes everything a Brewfile omits. Homebrew 7 also refuses a formula from an untrusted tap.

## Decision

`brew bundle` is the install and upgrade engine for every method it covers, in one call per request with the Brewfile on stdin. Uninstall is each method's native command, in the same method table, so there is still one dispatch. Methods bundle has no extension for (`bun`, `mise`, `script`, `shell`, `skill`) are hand-written rows of that table. A catalogue entry marked `trusted` is the owner's act of trust in a tap (pulumi, hashicorp, inferrs). `station tools upgrade` and `--via` upgrade catalogue tools only.

## Consequences

- Packages a manager holds that the catalogue does not name are no longer upgraded by station.
- One table is the whole dispatch; adding a method is adding a row.
- A trusted tap is visible in the catalogue and reviewed like any entry.
- 2026-11-06: mas left the bundle. Inside it mas's output is held until it ends, so a signed-out App Store hung the whole bundle, setup included, with no word. mas installs and updates by itself, one app per run, and the 60 s it may stay silent is how station reads a signed-out App Store.
