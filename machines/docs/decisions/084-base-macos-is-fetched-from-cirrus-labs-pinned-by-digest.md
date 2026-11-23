# ADR-084: `base-macos` is fetched from Cirrus Labs, pinned by digest

**Status:** Accepted
**Date:** 2026-11-23

Accepted by the owner on 2026-11-23, on the owner's word.

Amends [ADR-051](051-a-machine-is-a-base-and-its-machinefiles.md).

## Context

ADR-051 decided that Machines builds every base, macOS included, and that no base is fetched only. The macOS half was never built: `base-macos` has no recipe, and `station images update base-macos` asked a registry that releases no macOS image. On a new Mac on 2026-11-23, no Spwn gesture could make a macOS VM: `machines create --image base-macos` said to update, the update said no version is released, and its hint led back to `station images`.

Cirrus Labs publishes macOS images for tart. `ghcr.io/cirruslabs/macos-tahoe-base` is the one the Machinefile already named, by its floating `latest` tag. Read anonymously from the registry on 2026-11-23, that repository has `latest` as its only tag: digest `sha256:65c3133a633fda8753cf1f71f2d55727792d534a2125b0032466563dcc8e58fd`, uploaded 2026-11-07, a 50 GB raw disk sent as 27.1 GB of layers, 4 CPUs and 8 GB of memory in its config. Its sibling `macos-tahoe-vanilla` tags each macOS version, and there `latest` and `26.6.2` are one digest, uploaded 2026-08-18, before the base. So the base is macOS 26.6.2, the version ADR-051 names. That version is read from the registry, not from a booted guest.

## Decision

The owner decided on 2026-11-23: for now `base-macos` is fetched from Cirrus Labs' public tart images, not built here. This replaces ADR-051's "Station builds every one, macOS included: no base is fetched only" for `base-macos` alone. `base-ubuntu` and `base-debian` are still built on each Mac.

1. **The upstream is pinned by digest.** The `base-macos` Machinefile's `vm.image.from` names its image by digest, never by a floating tag: `ghcr.io/cirruslabs/macos-tahoe-base@sha256:65c3…58fd`. The pin changes only by a commit, as ADR-051 asks of every upstream.
2. **The version follows the image ADR-051 names.** The Machinefile's `system.version` is the pinned image's macOS version, `26.6.2`, and that version names the fetched image (`base-macos@26.6.2`). A new pin moves both in the same commit.
3. **The fetch goes through Spwn's own path.** `station images update base-macos` fetches the pinned digest, keeps it as an image disk in Spwn's tart home and writes its record, with the pin as its upstream. This is the same fetch, clone and record a released image takes, so `station images` lists it with its version and size, and `machines create --image base-macos` clones it. A fetch-only image is updated from its pin; every other image is updated from the registry Machines releases to.

## Consequences

- Trusting the image means trusting a third party. What a macOS workspace runs first (its user `admin` with a known password, a passwordless sudo, Homebrew, a CI runner's tools) is whatever Cirrus Labs put in it. The digest makes that content fixed and checkable, but nobody here reviews it. A pin moves only by a commit, after someone reads what changed.
- The build-here half of ADR-051 comes back when a macOS recipe exists. That means the plumbing that `linux.sh` gives Linux (the user, the image key, sshd with password login off, the units), run over this upstream or over Apple's IPSW, plus a gate that proves it on a macOS host. Then `base-macos` gets a `Recipe`, `images build base-macos` builds it, and this record is superseded.
- `latest` may move at Cirrus Labs while the pin stays: a Mac fetches the same bytes until a commit re-pins them. A re-pin at the same macOS version keeps its version too, so the next fetch on a Mac that already holds that version happens only after `station images delete base-macos`.
- The first fetch costs 27 GB of download and 50 GB on disk. Tart's pull cache is discarded once the image disk is kept.
