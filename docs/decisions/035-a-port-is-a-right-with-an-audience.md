# ADR-035: A port is a right with an audience

**Status:** Proposed
**Date:** 2026-11-11

Amends [ADR-018](018-two-port-classes.md).

## Context

[ADR-018](018-two-port-classes.md) shares every port a workspace listens on through Serve, filtered by two preferences (`min_port`, `ignored_ports`), and pins two dev-server profiles (Expo, Vite) at create. A Machinefile now says what a machine carries, and its `ports` key (`open`, `to`, `except`) says what it may open and to whom. Sharing everything that listens, to one audience, cannot express a database kept on its Mac, a port meant for the internet, or a port that must never leave. Tailscale Funnel, the one route to the internet the Mac already has, answers on 443, 8443 and 10000 only.

## Decision

A machine's ports are rights, never an inventory. Station observes what listens; a port inside the rights is shared with their audience on a port of the Mac it picks (sticky, as the reserved ports are), and one outside them is flagged (`listens, not allowed (open: 3000-3999)`), never shared until the owner shares it. The audiences are three, behind one contract, `ports/contract.Audience`, whose adapter is `tailscale`:

- `local`: its Mac alone, the loopback listener with no serve.
- `private`: your devices on the tailnet, Tailscale Serve (ADR-013's mechanism).
- `public`: the internet, Tailscale Funnel; it takes an explicit list of at most three ports, never a range, mapped in order onto 443, 8443 and 10000. With HTTPS links by path, every private share is a path of 443 and Funnel opens a port whole, so path mode never Funnels 443: the list maps onto 8443 and 10000, a third port is refused, and a private path is refused while 443 is funneled by anyone else.

Until a VM records the Machinefile it was made from, the engine reads the one named after it, at each run of the VM and at its adoption by an engine that finds it running. Rights are re-judged at every such read: a reservation records who made it (`by: owner` for the owner's `ports share`, `by: rights` for the rights; one made before this key says none, and reads as the owner's on a server, where only `ports share` shared a port, and as the rights' on a workspace), and one the rights made that they no longer allow is unshared and released, Funnel entry included, at once rather than after its idle days; one whose audience or Funnel port changed is shared again with the new one; the owner's own is kept, private when the rights do not give it their audience. A guest is never the owner: its own `ports share` is judged by its VM's rights, refused outside them and made as theirs (`by: rights`) inside, so it cannot reach past `except` or a `local` audience. Station never replaces a tailnet entry it did not make, and every path that drops a reservation (a release, its idle days, a narrowing, its VM's delete, a mode switch) removes the entry it made, in the mode it was served in, whether this engine served it or found it at its start, never an entry at its port and path carried from another port of the Mac, which the owner put there; every change to what a reservation serves (its port of the Mac, its scheme, its audience, its Funnel port, the mode) removes the old entry the same way before the new one is served. A workspace whose Machinefile says no `ports` (every VM made before rights) gets `open: 1024-65535, to: private`, which is what `min_port: 1024` and `ignored_ports: [22]` shared; a server gets none, as before. A Machinefile the decoder refuses opens nothing. What a dev server needs to know of its share (Expo's and Vite's env) is its catalog item's `ports`, and the profiles go: the engine reserves such a port, inside the rights, at each run of its VM so the env holds before the server starts; nothing is pinned. `min_port` and `ignored_ports` leave `preferences.json` at `station init`; `range` keeps its meaning, the ports of the Mac station picks from.

## Consequences

- The Private/Public classes of ADR-018 stand as the words of who can reach a port; the wiring behind them is the audience's.
- A public range is refused at decode, and a Funnel port is held by one share at a time.
- A removal the tailnet fails is owed and tried again at each tick, and the port of the Mac its entry carries from is given to no reservation until it goes: a failed `tailscale serve` never leaves a stale Funnel entry carrying another share to the internet.
- A rights file edited takes effect at the VM's next run, or at the next engine that adopts it running, as its env does; a narrowed right closes what it no longer allows then.
- A non-default `min_port` or `ignored_ports` is not carried: the owner writes it as the VM's rights.
- A Mac taken back to a station older than this decision does not know a Funnel entry as its own: a public share that engine releases leaves its Funnel entry on the tailnet, until the next start of an engine of this decision removes it, as an entry no reservation justifies.
- The label a port is shown with ("probably Vite") is a hint read from the items, never a decision; the surfaces that print it are lot L4.2's.
