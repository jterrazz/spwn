# ADR-0xx: Versions are semver ranges, applied every night

**Status:** Proposed
**Date:** 2026-11-12

## Context

[ADR-041](041-the-lock-is-what-apply-installs.md) made the lock what `apply` installs and `upgrade` the one run that moves it, and kept a Machinefile's `version` as a prefix a method pins (`22` met by `22.11.0`). Three things did not hold. A pinned app never moved again: `upgrade` kept it whole, and mise's `upgrade` could not move a runtime the lock had installed at an exact version, since that exact version is what `mise use --global` wrote. An app whose own updater moves it (Claude, Ghostty, a browser) read behind and was updated by brew under its updater's feet. And the shared source was pinned by a 40-character commit the owner moved by hand in `preferences.json`, which `apply` refused until `upgrade` took it (`machinefile.source-moved`). The owner's design round of 2026-11-12 (section C) settled the model: semver, `latest` by default, applied every night, the lock kept as station's invisible record, no git ref and no commit hash for the catalog.

## Decision

- **A version is a range, in npm's words.** An app entry's `version` is read once into a range: nothing or `latest` holds every version, `'24'` is `^24.0.0`, `'24.11'` is `~24.11.0`, three numbers are exact, and npm's `^` and `~` forms are taken as written. A word that is none of them is refused at its line.
- **A method holds what its install carries.** mise holds any range by its prefix (`node@24`), every version by `latest` whatever narrower channel the catalog writes (node's `lts`), and a range keeps the vendor the catalog names (`java@temurin-21`); bun any range by its specifier, apt and a release one exact version, every other method only its newest; npm and uv stay there while their installs ride brew's bundle. A range a method cannot hold is refused by the resolution, saying what the method holds and the way out, never faked; brew holds a major only by a versioned formula, and an item whose formula is one (`postgresql@18`) holds that major (`'18'`) and no other.
- **upgrade moves each app as far as its range lets it.** A method that holds any range installs the app again at it, whose newest it then holds; one exact version of a method that holds no range around it is kept; every other app is updated as its manager does. A newer version past the range (a major past `24`, a minor past `24.11`) is said waiting, with the line that takes it (`change '24' to '25'`), where the method reads the newest its manager offers; the move itself stays the owner's edit.
- **An app that updates itself is only checked.** A catalog item says `updates: self`; station reads it present, never behind, and never updates it. Its lock line keeps the version it was installed at: the lock records what is there, so a new Mac starts from it, and station never moves it. The catalog marks every cask Homebrew says `auto_updates`.
- **The lock stays, and its file stays `station.lock/v1`.** What `apply` installs is what the lock names (ADR-041's invariant): an upgrade relocks what it moved.
- **The engine applies it every night** on a Mac whose Machinefile is bound: at the first 03:00 of its clock past the last night that completed, or past the binding and the engine's start when none has, so a Mac asleep at that hour catches up when it is back and a Machinefile bound in the day waits for the next night. The managers first, then the apps, only the steps that ask no password (a root step waits for the owner's run); a night the engine's stop interrupts runs again at its next start. Its outcome is recorded in the ledger, station's memory, and read by the apps line of the Overview and the menu bar: `N moved last night`, and one Needs you line per version waiting.
- **No commit for the shared source.** `preferences.json` names it by `name` and `url` alone, and it follows its default branch: cloned at it, read where its clone stands, and moved to the branch's newest commit by `upgrade` (so every night) before anything is read. The lock records the commit each run read and refuses no run for it: `machinefile.source-moved` is gone. This replaces the pin of [ADR-033](033-a-machine-is-declared-by-its-machinefile.md) and the shared source's commit of ADR-041; the catalog's own version is station's.

## Consequences

- `pinned 22: kept` is gone for mise and bun: their apps move within their range at every upgrade, and the lock follows. apt's and a release's exact versions are still kept.
- A newer major is said only where a method reads its manager's newest (mise's `mise latest` today); elsewhere it is unsaid, never guessed. bun, npm and uv gain that read with their own lots.
- npm and uv cannot hold a range until their installs leave brew's bundle or the bundle carries a version (TODO).
- A change pushed to the shared source's default branch, a `script` method's included, runs on every bound Mac at its next night with no gesture of the owner's: the source is trusted as the owner's own store, and a clone that holds a change is still refused.
- An older `preferences.json` that names a `commit` is read with it ignored.
