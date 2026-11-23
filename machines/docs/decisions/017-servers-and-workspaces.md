# ADR-017: A VM is a server or a workspace

**Status:** Proposed
**Date:** 2026-11-06

**Amended by:** [ADR-078](078-a-server-is-set-up-from-its-machinefile-on-the-owners-verb.md): station sets a server up from its Machinefile, at its creation and on the owner's `upgrade --on` alone.

Written 2026-11-06 from the owner's round 11 to 13 decisions (K1, decisions 13 and 14) and the code that carries them.

## Context

v2 asked a lifetime at every create, temporary or lasting, and a lasting VM could be supervised. The owner's infrastructure was to move from OrbStack to a tart VM on a Mac mini: a VM kept always on, set up by its own tooling (Ansible), where station must keep it alive and check its health but set nothing up inside. A lifetime alone could not say that: what station does inside a VM depends on who owns its inside.

## Decision

A machine's kind is one of three: `mac`, `server`, `workspace`. A workspace is set up by station, for the owner and the agents: apps, settings, grants, ports. It is Throwaway (deleted when it powers off from inside) unless made `--permanent`, and started and stopped by hand. A server (`machines create --server`) is always on: it starts with its Mac's engine, boots again after a crash with a wait that doubles from 5 s, and is checked by its four health steps (Running, Has an address, SSH answers, Check passes, the last its `--check`). station sets up nothing inside a server: its sections are Overview, Activity, Ports and SSH, and Apps, Settings, Disk, Accounts, Background and Awake are refused on it as not for this machine. Stopping or deleting a server asks its name typed.

The kind is the first axis of `core/section.For`; the system (`macOS`, `Linux`) is its own word. The VM says it powers off with `station-intent=poweroff` on its console, unchanged from ADR-008.

This supersedes [ADR-008](008-every-vm-has-a-lifetime.md).

## Consequences

- A create needs no question: a bare create is a throwaway workspace.
- A failing health check never restarts a server: what runs inside is its tooling's; the owner is told.
- A server's own Tailscale device, if its tooling made one, is shown and never touched.
