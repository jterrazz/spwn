# ADR-056: Station names the Local Network approval a Mac withholds

**Status:** Proposed
**Date:** 2026-11-18

Amended 2026-11-24: H5 was false, so the dial alone is no longer the block (below).

## Context

macOS lets a process reach its Mac's local network only once that Mac has approved it for Local Network, and the
engine's VMs sit on that network (`192.168.64.0/24`). On a Mac that has not approved station, every guest dial fails
`EHOSTUNREACH`, so a create said `linux-2 is kept, failing SSH answers` and nothing else: nothing told the owner that
one click on that Mac's screen was the whole fix, nor which Mac it was. macOS asks only when a process first touches the
local network, and an engine started over SSH has no one at that Mac's screen to see the question.

Five facts the design leans on are HYPOTHESES, not yet proved on a real Mac:

- H1: one mDNS datagram to `224.0.0.251:5353` is refused with `EHOSTUNREACH` while the Mac has not approved station,
  and macOS asks the owner on its screen at that refusal.
- H2: an approved Mac sends it without an error.
- H3: an approval the owner declined fails the same way, so station cannot tell declined from not asked yet.
- H4: an ad-hoc signed binary is asked again after each upgrade.
- H5: on an approved Mac, a booting guest's dial fails `ECONNREFUSED` or times out, never `EHOSTUNREACH`. False on 2026-11-24: a MacBook the owner had approved logged `dial tcp 192.168.64.239:22: connect: no route to host` for development-linux.

## Decision

The engine reads the approval itself and every surface names the Mac:

- The engine reads the approval at its start, before its socket opens, bounded to a second, and again every 30 s while
  the Mac has not approved station. A refusal in its first 90 s reads `checking`, read again every 2 s, since macOS
  refuses the first read of a bundle it has just seen replaced ([ADR-073](073-spwn-app-carries-the-engine-on-a-mac-with-a-screen.md)'s
  gate). Its hello carries it, `local_network`: `approved`, `not-approved`, `checking`, or absent where it cannot tell
  (off a Mac, a read that does not end). The read is one adapter (`approvals/adapter/macos`, the
  approvals family's probe, [ADR-105](105-macos-approvals-have-one-owner-the-engines-approvals-table.md)), so its
  trigger changes alone if H1 or H2 fails.
- A guest dial a Mac refuses with `EHOSTUNREACH` is the block only when the engine's read of the approval, taken at
  that dial, says not approved. The engine keeps the VM `blocked: local-network` until a dial connects or its run
  ends, and answers a run it cannot reach `local_network`. With the read approved, checking or unable to tell, the dial
  is no block: the VM says the dial's own error and its address, and no cause. The engine knows no other read of the
  approval than that datagram; whether macOS exposes one is a HYPOTHESIS left unproved.
- `station engine start` waits 5 s at most for a `checking` read to settle, then exits 0 with the row `◐ local network
needs you · approve station on <mac>'s screen, or in System Settings > Privacy & Security > Local Network`, or
  `◐ local network checking · if macOS asks, <the same fix>` while the read still checks: the engine runs, and a VM created meanwhile is started once approved, its first boot never having run ([ADR-062](062-a-failed-create-stays-on-the-record-until-its-start-or-its-delete.md)).
- `machines create` says `<vm> is kept, failing SSH answers: <mac> has not approved station for Local Network`, its
  hint the same fix; `machines` lists the VM unreachable and the Mac as needing a look, both with that fix.

The words say "approved" and "not approved", never "allowed" or "denied" (H3).

## Consequences

- The owner learns the one gesture that unblocks a Mac, on whichever surface he reads, and from any Mac.
- The proof is owed on a real Mac once (H1 to H5): an `engine start` over SSH, its row and macOS's question; a create;
  the approval through Screen Sharing; the VM connecting by itself. A false H5 means the block is named only after two
  dials 10 s apart; a false H1 moves the trigger to the default gateway, in the same adapter.
- If H4 holds, every release asks again, which moves the signing decision forward.
