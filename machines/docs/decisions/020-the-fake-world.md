# ADR-020: Use cases are proved over a fake world

**Status:** Proposed
**Date:** 2026-11-06

Written 2026-11-06 from the owner's round 3 test policy and round 13 decision Q3, and the code that carries them.

## Context

The v2 specs drove the real binary against stub programs and asserted the argv each stub received. They proved that a command spoke its program, not what the owner got; a story spanning two Macs, a VM that crashes, a clock that moves or a battery that drains could not be told. The owner asked for tests of value only, in the owner's words, and every use case proved.

## Decision

`src/world` is one serialisable value: each Mac and VM with one slice per family (its apps, files, jobs, listeners, facts, power, vault, Tailscale node, engine), the image registry, and the scripted faults. Every contract has a fake in its family's `adapter/fake/`, over its slice. The spec build (`-tags world`, never shipped) wires the fakes instead of the real adapters when `$STATION_WORLD` names a world file, and steps each Mac's engine over the world's clock. A use case is told in `specs/usecases/<family>/<verb>/<outcome>.spec.yaml`: seed a world, run the real CLI, assert what the owner reads, and read the world back through the CLI. A world event (`station world <event>`, hidden, the spec build's only) moves the world between runs: a clock that advances, a VM that crashes, a battery that drains. An adapter's own argv is proved once, in `specs/adapters/<tech>/`, against stub programs. The rule `coverage` requires a story of each outcome every call declares; `spec-build-never-ships` proves the release binary links no world and no fake.

## Consequences

- A use case says nothing of tart, launchctl or brew, so a second hypervisor passes the same stories (`make test` runs the VM stories again over the fake `vz`).
- A fake and the real adapter pass the same contract suite, so a fake cannot drift from what it stands for unnoticed; a fact the fake keeps in memory and the real store drops is proved at the real store.
- Two slower tiers run real VMs of this Mac (`specs/machine/`), outside `make check`.
