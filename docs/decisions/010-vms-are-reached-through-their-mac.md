# ADR-010: VMs are reached through their host

**Status:** Superseded by [ADR-016](016-api-v2-and-remote-by-api.md)
**Date:** 2026-11-01

## Context

Rounds 2 to 4 put every VM on the tailnet: a single-use auth key minted at create through an OAuth client, a tag and its ACL lines in the owner's policy, Tailscale SSH inside the VM, and a userspace tailscaled station ran on each host. It worked, at a price: a secret per VM, an admin action before the first create, a node to remove at every delete, and a daemon of station's own next to the owner's Tailscale.

## Decision

A VM is never on the tailnet. It is reached through its host: its `~/.ssh/config` block runs `station machines pipe <vm>`, on its own host directly, from another host over one ssh hop. The VM authorizes its host's keys in a managed block of its `authorized_keys`. Each host runs the Tailscale app, which the owner logs into; station only reads it (`tailscale status`, `tailscale serve`) and drives nothing else. Between hosts there is ssh only, Remote Login and a key; no network API until an app needs one. The owner's Tailscale policy needs no line for station.

## Consequences

- The VM-join client, the minted keys, the VM tag, the node removal, the userspace tailscaled and the `tailnet up`/`down` verbs are gone; `station setup` retired what an earlier round left, until station dropped every path that upgrades an older install.
- `ssh <vm>` works while the daemon restarts: `machines pipe` needs only the VM's current NAT ip.
- What a VM serves reaches the tailnet only through its host's publication ([ADR-012](012-ports-are-reserved-and-shared.md)).
- An iPhone on the Wi-Fi reaching a VM directly needs a bridged macOS VM, which stays unproven.
