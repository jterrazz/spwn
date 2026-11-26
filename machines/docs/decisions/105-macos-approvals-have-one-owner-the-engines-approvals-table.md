# ADR-105: macOS approvals have one owner, the engine's approvals table

**Status:** Proposed
**Date:** 2026-11-26

## Context

Spwn needs consents only macOS gives: Local Network for every dial to a VM and to a Mac nearby, a login item, notifications, Touch ID. Each was read where it was first needed, in its own language: the Local Network datagram in `wire`, its timing in `engine/process`, its words in `machines`, the engine's login job read in `platform/darwin`, the window's login item in its Rust shell, the menu bar's notifications in Swift. So no surface could say, in one place, what Spwn has and what it lacks, and the owner was told, or not, by whichever part met the refusal first. He asked for one onboarding, the same list in Settings, and the code that knows each consent in one place, so he is told once (2026-11-26).

"Permission" is already a file's mode in the catalog (`permissions`), "grant" is the vault's and "need" an app's; "approve" is the Local Network word the words already use. The owner chose the name "Approvals" (2026-11-26).

## Decision

- **One family owns every approval: `approvals`.** An approval is macOS's consent to Spwn, which Spwn can ask for and read but never give itself. The family holds the closed table (`approvals.ID`): each row's sentence, whether Spwn needs it, the closed states it reads in, and one probe per approval behind its contract, a macOS adapter and a fake over the world. No file outside the family reads an approval: the rule `approvals-owned` refuses the probes' names elsewhere, the cgo files of `platform/darwin` it will wrap aside.
- **Local Network is its first row.** Its probe is the mDNS datagram ADR-056 chose, moved whole into `approvals/adapter/macos`, and the age of a refusal (`checking` in the engine's first 90 s, read again every 2 s, then every 30 s while refused) is the family's rule, which the engine calls. Its states are `approved`, `refused`, `checking` and `unknown` (a read that cannot tell). The engine's hello keeps its `local_network` words until every Local Network sentence comes from the family.
- **`approvals.list` is a view of `engine.state`** ([ADR-070](070-the-engine-keeps-one-read-model.md)): each row carries its id, state, holder (the app that carries the engine, or the bare binary), whether it is required, its sentence and its fix, the pane once macOS asks no more. `station approvals` prints it, one page under ADR-021's noun grammar.
- **Everything can be skipped** (the owner, 2026-11-26): no approval stops Spwn from running; a missing one is said by "Needs you" and by its row.
- **macOS asks only after a press.** Every prompt Spwn causes follows a gesture of the owner's: `approvals.ask` comes with the first-run Guide, and the sentence macOS shows for Local Network is the bundle's own `NSLocalNetworkUsageDescription`, "Spwn reaches the VMs it runs on this Mac and your Macs nearby." (the owner, 2026-11-26).

## Consequences

- The approvals the app holds (open at login, notifications) join the table once Spwn is one app with one identity; until then "once" cannot hold, since macOS charges three names.
- An engine started over ssh can never be approved; its row will say so (`cannot-ask-here`) and name the gesture that gives an approved engine.
- The Guide reopens by itself only when a required approval is lost, at most once per app version; every other loss is a "Needs you" finding and its row.
- [ADR-056](056-station-names-the-local-network-approval-a-mac-withholds.md)'s read is this family's probe; this record amends it, and the deferral of its first read to a press on a fresh Mac will amend it again.
