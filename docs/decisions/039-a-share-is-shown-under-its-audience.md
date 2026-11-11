# ADR-039: A share is shown under its audience

**Status:** Proposed
**Date:** 2026-11-11

Builds on [ADR-035](035-a-port-is-a-right-with-an-audience.md).

## Context

[ADR-035](035-a-port-is-a-right-with-an-audience.md) gives a share one of three audiences, `local`, `private` or `public`. The surfaces still drew every share under Private, "shared with you on Tailscale", and `station.ports.list/v1` named every shared port `class: private`, so a port open to the internet read as a tailnet one, and a port kept on its Mac as reachable from the tailnet.

## Decision

- The CLI list and the menu bar's Ports page place a share in the section of its audience: Public shares, Private, This machine only; a listener outside the rights is not a share, and stays with the listeners, flagged, with the owner's Share.
- `station.ports.list/v1` stays v1: it gains `to`, the audience of a shared port, and `class` becomes `public` when `to` is `public`, as it already was when what listens there is reachable outside the tailnet or declared public. No key is removed or renamed, so a reader of v1 keeps reading it; a `local` share keeps `class: private`, the closed union of two classes, and `to` says the rest.
- A row says what probably serves its port, from the catalog's labels, as a hint.

## Consequences

- A v1 reader that took every shared port as private now sees `public` on a funneled one: the fix is the point, and it is the one value that changes meaning.
- The menu bar gains a card, This Mac only, shown only when a share uses it; no Swift changed, the card is a section of the page.
- A `local` share is `private` in `class` and `local` in `to`: a v2 would give it a class of its own, and is not taken for one value.
