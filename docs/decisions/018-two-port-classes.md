# ADR-018: A port is Private or Public

**Status:** Proposed
**Date:** 2026-11-06

Amended by [ADR-035](035-a-port-is-a-right-with-an-audience.md): which ports are shared, and with whom, are a machine's rights.

Written 2026-11-06 from the owner's round 12 decision P1 and the code that carries it.

## Context

v2 shared every port a VM listened on through Tailscale Serve and said nothing of the rest. Measured on the Mac mini, other software listened on every interface of the LAN, reachable by any neighbour, and nothing in station showed it. The owner wanted to see who can reach each port, with as few classes as tell the risk.

## Decision

Every listening TCP port, on a Mac or in a VM, is in one of two classes: **Private**, reachable only through Tailscale (a workspace's port shared by its Mac's engine on a reserved port of the Mac, through Serve; ADR-013's mechanism, unchanged), or **Public**, reachable from anywhere else, the internet or the LAN. A public port is declared, with its route (`station ports declare <port> --via cloudflare-tunnel|tailscale-funnel|lan`), and the route is verified at every list through the `PublicRoute` strategy. A port reachable outside Tailscale that nobody declared is a red finding, and the engine's `exposure` job tells the owner when one appears. The declarations live in `~/.config/station/public.json`; the reserved ports in `~/.local/state/station/leases.json`.

This supersedes [ADR-013](013-every-port-leaves-through-serve.md), whose sharing it keeps as the Private class.

## Consequences

- There is no "local network" class: the LAN is Public, so a port open on the LAN is either declared or a finding.
- Declaring is a statement, never an effect: station opens no route; it reads whether the declared one reaches the port.
- `station ports share` and `stop-sharing` keep ADR-013's reservation, sticky on (VM, guest port).
