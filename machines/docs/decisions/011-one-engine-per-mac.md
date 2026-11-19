# ADR-011: One daemon per host

**Status:** Superseded by [ADR-016](016-api-v2-and-remote-by-api.md)
**Date:** 2026-11-01

## Context

Round 3 gave each VM a launchd agent running its own supervisor. A host with five VMs ran five supervisors, none of which could hold what spans VMs: the ports their services are published on, one view of the fleet for a client that is not the CLI.

## Decision

Each host runs one `station daemon` under one launchd agent, `com.jterrazz.station.daemon`, installed by `station setup` once tart is there. It runs each VM detached (a wrapper writes the run's exit code to a file), so a daemon crash, upgrade or restart never stops a VM, and at start it adopts the runs still alive. It reads each run's end with round 3's rules, unchanged: a temporary VM's declared poweroff deletes it, a reboot never does, a lasting VM restarts after a crash, four stops in a minute leave a VM broken. Its state is served on the local socket `~/.local/state/station/daemon.sock` in the JSON-lines protocol `station.daemon/v1`; the CLI is a client, and a menu-bar app would be another. With the daemon down, reads degrade, `create`, `start` and `restart` refuse, `stop` and `delete` act directly.

## Consequences

- The per-VM agents and `machines supervise` as a supervisor are retired; setup migrates them, refusing while one of their VMs runs.
- One process per host owns the leases and the ssh connection to each VM's guest ([ADR-012](012-ports-are-reserved-and-shared.md)).
- The protocol is versioned: a client of another version is refused `unknown_op`, never misread.
