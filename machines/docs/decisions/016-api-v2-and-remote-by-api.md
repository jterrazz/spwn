# ADR-016: Every machine is reached by its engine's API

**Status:** Proposed
**Date:** 2026-11-06

Written 2026-11-06 from the owner's round 9 and 13 decisions and the code that carries them.

## Context

In v2, `--on mac-mini` executed `station <words>` again over ssh and relayed its text, All machines merged the JSON each Mac printed, and a VM was reached by `ssh` through its Mac's `station machines pipe`. The far side rendered, so another machine's page could differ from this one's; each command had its own way to a VM; and the engine spoke a protocol of its own (`station.daemon/v1`) beside the CLI.

## Decision

A use case is a call with a name (`machines.stop`), and every caller reaches it through `core/call.Router`. `remote.Router` decides where it runs from the call's `Where` and its target: in this process; at the target, here when the target is this machine; or in a Mac's engine. This Mac's engine is reached over its unix socket (`remote/adapter/unix`), another Mac's by `ssh <mac> station engine attach` (`remote/adapter/sshstdio`), which bridges ssh's stdio to that Mac's socket. A VM is reached through its Mac's engine, which relays the frame over its one guest connection to `station guest serve --stdio`. Between machines the frames are `station.api/v2`, one JSON object per line: `hello`, `call`, `watch`, `progress`, `out`, `error`, `done`. `make generate` projects the registry into `src/remote/api/calls.json`, the list a client such as the menu bar reads.

The far side returns data, never text; this binary renders it with its own pages. `Hello` carries the far version and the calls it serves, so a call the far side does not serve is the error `far-too-old` with the command that updates it. A confirmation travels in the call's input and is judged by the far side with the same `confirm.For`.

This supersedes [ADR-010](010-vms-are-reached-through-their-mac.md) (VMs through their Mac by ssh) and [ADR-011](011-one-engine-per-mac.md) (one daemon per host, `station.daemon/v1`); what they decided of the network stands: a VM is never on Tailscale, between Macs there is ssh only, and the engine listens on no TCP port.

## Consequences

- `station apps --on linux-1` prints what `station apps` prints on linux-1, in every form, and the TUI and the menu bar work on any machine by changing a target.
- All machines is a fan-out of calls (`remote/each`), 4 at a time with a budget each; a Mac that does not answer is a finding, never a failure.
- A VM needs no key to its Mac: its few engine calls ride the socket the engine offers inside it over the same guest connection.
- The managed `Host` blocks of `~/.ssh/config` stay for the owner's own `ssh`, editors and Herdr; station's code never runs `ssh <vm>`.
