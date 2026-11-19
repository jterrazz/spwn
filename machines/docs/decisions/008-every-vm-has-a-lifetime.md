# ADR-008: Every VM is temporary or lasting

**Status:** Superseded by [ADR-017](017-servers-and-workspaces.md)
**Date:** 2026-10-31

## Context

Where a VM runs matters less than whether it is meant to last. A temporary VM should vanish with its run; a lasting one should survive. Virtualization.framework tells the host only that a guest stopped, and `tart run` exits 0 for a poweroff and a stop alike, so the host cannot tell a reboot from a poweroff.

## Decision

Lifetime is asked at every create and never defaulted. The VM says when it powers off: a unit in the image writes `station-intent=poweroff` to its console, which the VM's supervisor reads. A temporary VM is deleted by that poweroff, by `delete`, or by `clean stale-vms`; a reboot, a host reboot or any stop station cannot read as a poweroff never deletes it. A lasting VM stays until deleted. The catalogue is place-aware, so each tool and item says which kinds of machine it fits.

## Consequences

- A reboot inside a temporary VM is safe.
- If the console cannot be read, a poweroff leaves the VM stale for `clean stale-vms`; nothing is deleted on a guess.
- A temporary VM cannot be stopped or restarted from the host, only deleted.

Round 5 (2026-11-01, [ADR-011](011-one-engine-per-mac.md)): the host's one daemon, not a per-VM supervisor, reads the intent.

Round 13 (2026-11-04, the v3 specification's SP-4 and SP-5): the words are throwaway and
permanent, and two consequences above no longer hold. A workspace never starts by itself,
not even a permanent one with its Mac (only a server comes back); and a stop from station
keeps a throwaway workspace, stopped with its disk, so it may be stopped and started. A
throwaway is destroyed only by `machines delete` or by a poweroff from inside it.
