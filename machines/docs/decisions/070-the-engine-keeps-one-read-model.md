# ADR-070: The engine keeps one read model; pages and views are its projections

**Status:** Accepted
**Date:** 2026-11-21

Accepted by the owner on 2026-11-21.

## Context

A page of the menu bar was read only while a client watched it: each subscriber ran its own feed, whose first frame was an empty skeleton and whose reads ran one after another, so a machine's page filled after the sum of six to eight reads, up to a far Mac's 5 s budget twice. The window asked every read again every 30 s, on every focus and after every gesture, each a fleet read over ssh, and drew "Reading" over the page in between. The two apps shared nothing but the socket, and judged the engine's version by two rules: the menu bar called an engine older than its pages "stopped", the window called it "running". The lanes pool kept some readings for the engine's whole life, until a `Fresh` read forgot them all.

The owner decided on 2026-11-21: the engine keeps the machine pages' data warm permanently; both apps share one read model and one compatibility verdict; a far Mac pushes its slice when its record says so, else it is polled about every 30 s; with no client attached the fleet is read every 60 s; the window shows after its first paint, capped at 0.8 s, and its no-engine panel only after about 10 s.

## Decision

- **One store.** The engine holds one snapshot per revision of slices by key (`mac:<m>`, `ports:<m>`, `overview:<m>`, `awake:<m>`, `activity:<m>`, `signins:<m>`, and This Mac's own `tailnet`, `records`, `vault`), each with its value, its hash, when it was read and whether it is fresh, stale or unread (`apps/engine/engine/store`).
- **What is not kept warm.** A machine's apps, settings and disk, and the two matrices, are not slices: kept permanently they would read brew, apt and the settings on every machine forever, a cost the owner's decision, which names the machine pages, does not cover. The window keeps reading them by their calls until benchmarks measure them and a later decision keeps them.
- **Readers, not watches, read.** Each slice has one reader with a cadence (attached and idle), a budget, and the gestures that make it due early (`apps/engine/engine/readers`). This Mac and its fleet are read every 30 s while a client is attached and every 60 s when none is; the overview every 15 min and after a changing call; sign-ins every 5 min attached, 15 min idle. A read that fails or runs late keeps its value, marked stale.
- **Pages and views are projections.** A page of the menu bar and a view of the window are pure functions of a snapshot, computed once per revision for every subscriber; a watch's first frame is the current projection, never a skeleton, and no read runs on an opening. A page of moving numbers (the live keys, the Activity page) stays on its 2 s sampler. `engine/feeds` imports no call (the architecture rule `FeedsReadNoCall`).
- **One stream, `engine.state`.** A Registry watch served by the engine: `opened`, then a `snapshot` (or, given `since` within the last 256 revisions of the same engine, `changed`), then `changed` with the whole value of each key whose hash moved, coalesced over 250 ms, and a `tick` every 15 s, which is the stream's liveness. A key is the unit of change; values are never patched.
- **Far Macs read their own store.** This Mac copies another Mac's slices from that Mac's `engine.state` over the link the Router already holds. Push or poll is a choice of each Mac's record in `preferences.json` (`macs.<name>.pushes`), and poll is the default: a held watch for a Mac whose record says `pushes`, else one ask every 30 s attached and 60 s idle. No far Mac is dialled on an opening. A far engine without `engine.state` is `far-too-old`, never read by the older calls. The Mac's record is remembered from what it sent.
- **One verdict.** `opened` carries the engine's page and state revisions and `fit` (`fits`, `engine-older`, `app-older`), computed by one function, in the same words for both apps. A client declares the range of revisions it draws: the newest it knows and the oldest it accepts. The verdict is `app-older` only when the engine's revision is above the newest the app knows, `engine-older` only when it is below the oldest the app accepts, and `fits` anywhere in between. A stream that does not fit sends nothing after `opened`; a client draws nothing before a `fits`, and an `opened` without `fit` is an engine older than the verdict.

This amends [ADR-016](016-api-v2-and-remote-by-api.md) (a far Mac's data now also arrives as a held watch on its link) and generalises the engine's rule that a reading slow or fanned out to every Mac is kept by a job, never read inside a watch. It reverses the rule that a page is read only while a client watches it.

## Consequences

- Opening a machine's page or the window's frame costs no read; two subscribers cost one read per slice.
- The engine reads all the time: every Mac reads itself every 60 s with no app open, every 30 s with one, and each Mac is asked by every other at that pace (or pushes). That is the cost the owner accepted for warm pages.
- A silent Mac keeps its last values, drawn stale, instead of losing its rows or turning its VMs to unknown.
- The menu bar's and the window's verdicts can no longer disagree; the menu bar no longer says "stopped" for an engine that runs. A newer app keeps drawing an older engine down to the oldest revision it accepts, so an app update does not wait on every engine of the fleet.
- A slice is encoded as an answer is, so a secret is withheld before the store holds it.
- The window's 30 s re-ask, focus re-ask and hello polling end with its adoption of the stream; the window keeps calls for gestures and for what the store does not keep (apps, settings, disk, the matrices, the engine's job state).
- The CLI's `station machines` keeps its own read of every Mac; whether it reads the engine's view instead is not decided here.
- A Mac whose station predates `engine.state` is read as `far-too-old` until it is updated.
