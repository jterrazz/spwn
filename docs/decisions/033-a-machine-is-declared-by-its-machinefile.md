# ADR-033: A machine is declared by its Machinefile

**Status:** Proposed
**Date:** 2026-11-11

## Context

[ADR-031](031-the-catalog-defines-the-machinefile-chooses.md) made the catalog files: an item says how to install or set one thing on each system. What one machine carries had no file: `init` ends on none ([ADR-030](030-init-makes-a-machine-known-apply-carries-its-machinefile.md)), the apps of a machine are what the owner installed one at a time, its settings what was applied, a VM's resources the flags of its `create`, the accounts shared with it and the ports it opens what was asked call by call. Nothing could be read back, shared between machines, fixed locally or explained, and a catalog decoder that refuses unknown keys means a format extended later breaks every older station.

## Decision

A machine is declared by one file, its Machinefile, `station.machinefile/v1`, and station computes what to do from it, never from a list of its own.

- Its families of keys, in reading order: `machine` (what it is: `kind` computer, workspace or server, and its `system`), `vm` (how a VM runs: lifetime, resources, network, image; refused on a computer), `extends` and `prefer` (what it inherits, the managers it tries first), `apps` and `settings` (what it carries, with option values, a secret as a `vault:` reference), `accounts` (the accounts shared with it, and those it keeps), `ports` (what it may open, and to whom: a right, never an inventory). A file without `machine` is a mixin, only extended.
- v1 is whole now: every key the plan, apply, the lockfile, the ports' rights, images and the menu bar will read is declared and decoded, read or not, so a key read later never breaks an older station; the decoder is the catalog's, strict the same way, and its JSON Schema is projected by `make generate`.
- Files come in layers, strongest first: the owner's (`~/.config/station/`), one shared source (a git repository pinned at a commit, cloned read-only into the cache), the built-in ones. A same-name file of a stronger layer replaces a weaker one whole, an item as a Machinefile; the owner's overlay, `<name>.local.yaml`, changes a Machinefile without touching it, and is the one file a surface may write.
- The resolution is a pure model of the `machinefile` family: the files merged in extends order, each item's method chosen by selector, prefer and each system's priority, what an item requires and its manager brought in, the steps ordered by requires, managers and after; a missing method, an unpinnable version, a cycle, `vm` on a computer and a newer schema are plan errors. The family installs nothing: it calls `apps`, `settings`, `accounts` and `ports`.
- Every value says where it comes from: `station explain <key>` prints each value with its layer and the file or item that says it.

The format is [Machinefiles](../reference/machinefiles.md).

## Consequences

- The user catalog is read by every command: an owner's item replaces the built-in one whole, and a file the strict decoder refuses stops every command that reads the catalog at its line, as a broken `preferences.json` does; `--version`, help and the engine run on the built-in items and say the file is refused, so the menu bar keeps its engine.
- The shared source is remote code (a `script` method runs on the machine): it is read only at a whole commit, never a branch, and only the owner moves the pin. Until the lockfile holds it (L2.4), the pin is in `preferences.json`; only the Machinefile's calls read the shared source, every other command reads the user and built-in catalog.
- `station-linux` and `station-macos` are built in, so a machine with no shared source can extend one; the owner's own Machinefiles live in the shared source, never in the binary.
- The choices a Machinefile carries leave their command flags and preferences in the lots that read them (settings Off here, the ports' range, a VM's resources); until each lands, both exist and the old one is what runs.
