# ADR-051: A machine is a base and its Machinefiles, one base per distro

**Status:** Accepted
**Date:** 2026-11-17

## Context

Station knew two images: `station-linux`, built from its Machinefile with a bake list a gate
needs (mise, git, Docker, build-essential, the Chromium libraries), and `macos`, only fetched,
its sizes written in the code. A macOS workspace started from a Cirrus Labs image carrying
Homebrew, node, ruby and a CI runner, the menu bar's rig from an 83 GB Xcode image with
Flutter, the Android SDK and 21 simulator runtimes. The window made a machine from a
Machinefile, the menu bar from a hard-coded list of two image names: two models of what a
new machine can be. The owner asked for minimal images per system, Debian beside Ubuntu,
a lean Xcode, layers, and a picker that separates the image from what a Machinefile adds
(2026-11-17).

A mixin is never applied today. The word "base" named the upstream disk an image is built
from, the record's `base` field included. Measured on this Mac on 2026-11-17: an APFS clone
of a VM is made in 0.018 s and costs 36 KB until the guest writes; discard reaches a raw
disk's host file from a Linux and from a macOS guest; `tart list` leaks one kernel APFS
object per call while an ASIF disk is in the tart home, none with raw disks (tart issue
1427).

## Decision

The owner settled the model on 2026-11-17, by boundary rather than by measurement: "c'est
juste par frontière très nettes que je veux la distinction, ya des bases par distro et c'est
tout". The same day the owner accepted the lot plan's seven questions as recommended, the Xcode copy
of point 6 among them.

1. **The bases are three, one per distro: `base-ubuntu` (24.04), `base-debian` (13) and
   `base-macos` (26.6.2).** Nothing else is a base. Station builds every one, macOS
   included: no base is fetched only. Each upstream a base starts from is pinned by
   digest in the base's Machinefile and changes only by a commit.
2. **A base holds plumbing; everything else is a Machinefile.** Plumbing is what station
   needs to reach, run and apply a machine: the user, the key, sshd, the units, the
   console, the binary, the login shell. `clt` and `xcode` are Machinefiles, composed like `dev`, `go-dev`, `web-dev`, `containers` and
   `browser-tests`. Nothing rides in a base because most machines want it.
3. **A Machinefile may be baked into a layer that clones share**: a cache of that
   Machinefile over its base, never a base, named `layer-<machinefile>`. `layer-gate` is
   one (the old `station-linux` list, for the gates), `layer-menubar-rig` another
   (`base-macos` and `xcode`). A layer is baked when a first start is too slow for how often its machines are made. A layer whose
   base has a newer version is stale, and a base version a layer still holds is not
   deletable. A byte is deleted in the layer that added it, since a clone's delete never
   frees its parent's bytes.
4. **Layers are APFS clones of raw disks, for both systems.** No base, layer or VM disk is
   ASIF. Stacked ASIF overlays are a later storage format behind the same catalogue, once
   tart issues 1403 and 1427 are settled.
5. **A machine is a base and a purpose.** `machines.create` takes `Image` and `From`
   together when `From` names a purpose mixin; the machine's chain is the base's
   Machinefile, then the mixin, and its first start applies it. A full Machinefile in
   `From` still fixes its base, and `Image` beside it is refused. `From` empty is the base
   and nothing more. `machines.new` answers the bases, the purposes each base offers and
   the size presets; the window and the menu bar both draw from it, and neither has a
   toolchain row: `clt` and `xcode` are purposes.
6. **Xcode enters a machine as a copy of the Mac's own Xcode**, read through a read-only
   share, at the version the `xcode` Machinefile pins; no Apple account and no download
   enter a VM. Simulator platforms are never installed.
7. **The words follow the boundary.** `station-linux` is renamed `base-ubuntu` and `macos`
   `base-macos`, with no alias: an image built under the old names is no longer read. The
   disk a base is built from is its **upstream**, and the record's `base` field is
   `upstream`, so "base" names only the three per-distro images. "Layer" names only
   a baked Machinefile: where a value of a resolution was read, the resolution's old
   layer, is its **origin** (`machinefile.OriginKind`), renamed on the wire before the
   images are.

## Consequences

- A purpose costs its first start: one to five minutes on Linux, three to six on macOS
  with Homebrew. Gates keep a one-minute create through `layer-gate`.
- Homebrew leaves every base; tier 3's bare Mac is an honest bare Mac.
- A Mac whose Xcode is not the pinned version cannot bake `layer-menubar-rig`; the version and
  the menu bar's goldens move together, by a commit.
- A clone's private bytes are invisible to `du`: footprint is read by `df` deltas.
- The upstream is pruned once its base is built and pulled again at the next build.
- Debian enters the Linux machines: `os: debian` matches them, and every Linux package
  answers for its Debian names.
- A machine never moves to a new base version by itself; a persistent one moves by a
  recreate, with its work on a data disk once that lands.
- Ubuntu's minimal cloud image waits for a proven import (network, host keys, user
  injected before the first boot); the slimmed Cirrus Labs upstream serves meanwhile.
