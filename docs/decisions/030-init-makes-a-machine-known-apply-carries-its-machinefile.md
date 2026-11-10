# ADR-030: init makes a machine known; apply carries its Machinefile

**Status:** Proposed
**Date:** 2026-11-10

Supersedes, once accepted, the "one bootstrap verb" of [ADR-009](009-one-bootstrap-verb-and-no-values-in-argv.md); its rule of no values in argv stands.

## Context

`station setup` was the one bootstrap verb, and it grew into the whole install: the name, the `~/.zshrc` block and the XDG layout, then the engine, five "system apps" installed eagerly (Homebrew, mise, git through `xcode-select`, tart, Tailscale) and every setting that needed nobody. On a fresh MacBook (2026-11-10) that made one command that asked for sudo, ran Homebrew's installer and failed on its PATH, and a Ctrl-C at a cask's prompt read as a failed install. Most of what it installed is needed only by some command families, and which apps and settings a machine carries is a choice that belongs to the machine, not to the bootstrap.

## Decision

The bootstrap and the content split. `station init [--name <name>]` makes a machine known to station and nothing more: its name, the one managed `~/.zshrc` block, the XDG folders with every store's migration, and the apps' own files. It installs no app, runs no package manager and asks no password, and it ends on the empty state of what the machine carries, then the next steps. What a machine carries is its Machinefile, which `station apply` carries out (a later phase); until then apps and settings come one at a time, `station apps install <app>` and `station settings apply <setting>`, and the engine with `station engine start`, which installs its job the first time. `install.sh` runs `init` and passes its own arguments on. The Go family keeps the name `setup`, since Go names no package `init`; the call is `setup.init`. `station setup` stays a hidden alias of `init` for one release, saying so on stderr with where apps and settings went.

## Consequences

- The install is three steps with no sudo, and is safe on a machine without Homebrew; a fresh Mac that wants the engine types `station engine start`.
- The eager System list, setup's apps and settings steps and its engine step are gone; the Tailscale App Store wrapper is the setting `tailscale-cli`, named by the `tailscale` app's `then`, and the superseded formula goes with `apps install tailscale`.
- A Mac that relied on `setup` to apply settings or install apps now does it by name until the Machinefile lands.
- The word `setup` is retired from the interface; it survives as the family's package name and as `station.status/v1`'s `setup` key, kept byte for byte.
