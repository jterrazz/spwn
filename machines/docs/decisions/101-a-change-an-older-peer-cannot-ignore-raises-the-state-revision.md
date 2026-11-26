# ADR-101: A change an older peer cannot ignore raises the state revision

**Status:** Proposed
**Date:** 2026-11-26

## Context

[ADR-070](070-the-engine-keeps-one-read-model.md) gave the window and the engine one verdict on the revision of `station.state/v1` each speaks, and [ADR-072](072-before-1-0-every-station-runs-the-same-release.md) made it one constant on each side. Nothing said when the constant moves, so it never did: the tree of 2026-11-26 (ee5d40036) and the one of 2026-11-25 (68278fdd8) both said revision 1, yet the newer window subscribes to `engine.guides`, answers a question the engine raised with the name it asked about, and reads a need's `Pane` and `InTerminal`. Against the older engine it was judged fitting and broke: every view refused as `engine.state-unknown-key`, a workspace stop asked again and again, a System Settings button with no pane. The verdict page that says the engine is older never showed.

A rule left to memory is forgotten at the next view. A digest of the whole wire would be bumped by reflex at every field added, which is the change an older peer ignores best.

## Decision

- **A revision names what both peers must understand to talk at all**: the views the window asks in its one request, each confirmed call's forms, each call's `in` and `out`, every field of an answer a client reads as always there, and every enum member. A change to one of those raises the revision on both sides; anything else is additive and raises nothing. The table is [the reference's](../reference/api.md#what-raises-the-revision).
- **The rule is pinned, never remembered.** `make generate` writes `packages/api/revision.json`: the revision, and each of those things as it was first generated at it. A pin keeps its recorded value while the revision stands, so a regeneration absorbs nothing, and a test fails naming each pin that moved until the revision is raised and the file generated again. A new call or type is pinned when first generated, without raising anything.
- **The verdict comes before the keys.** An engine answers a client of another revision its verdict whatever keys it names, so a window that asks a view this engine does not know is told it does not fit rather than refused.
- **No peer is taught the other's revision.** A window does not adapt to an older engine, nor an engine to an older window: the verdict says which to update.

## Consequences

- Revision 2 is the first raised under the rule. An engine of revision 1 validates the keys before its verdict, so against it a window of revision 2 is refused, not judged; the window draws that refusal in the engine's words and asks again, and the verdict page shows from the next engine on.
- How a peer reads an answer it receives, as a confirmation's answer, is behaviour no schema carries, and is not pinned: such a change raises the revision by the author's judgement.
- An enum member added to an optional field moves its pin, since a member an older window does not know is no absence; a field the author means as additive is declared under `omitempty`.
- The menu bar speaks `station.page/v1`, whose revision is its own constant on each side and outside this rule.
