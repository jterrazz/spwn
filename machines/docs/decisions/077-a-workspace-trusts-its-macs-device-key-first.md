# ADR-077: A workspace trusts its Mac's device key first

**Status:** Proposed
**Date:** 2026-11-22

Amends [ADR-064](064-a-key-never-leaves-its-device.md).

## Context

ADR-064 §3 makes every block on a Mac offer the device key first, and says nowhere that a VM authorizes it. The guest was told only of the Mac's `~/.ssh/id_ed25519.pub` and of every line of its `~/.ssh/authorized_keys`, the copy ADR-010 started. On a fresh Mac there is no `id_ed25519`: ssh offered the device key, the guest refused it, and `ssh <workspace>` answered `Permission denied (publickey,password)` (the new Mac mini, 2026-11-22).

The copy is also what carries Mac-to-Mac reach today: a far Mac's ssh is end to end through the pipe, so the VM must trust the far Mac's own key, which only that Mac's `authorized_keys` line on this Mac says. Cutting the copy now would break it, and ADR-064 §4 keeps `~/.ssh/id_ed25519` opening every machine until the device book (lot L3).

The copy grants no power a login does not already hold: every line of a Mac's `authorized_keys` is a full login as the owner on that Mac, which reaches its VMs through the image key (`~/.local/state/station/images/key`, authorized in the guest as `station-image`) and through the engine socket. This rests on the image key file being owner-readable on a real install, which this ADR did not verify.

## Decision

A VM's managed block holds, in this order and each key once: this Mac's device key (the line of `~/.local/state/station/device-key.pub`, when this Mac has one), the Mac's `~/.ssh/id_ed25519.pub`, then every non-comment line of the Mac's `~/.ssh/authorized_keys`. The engine sends them in its hello and again on each walk when they change, so a device key made after the guest connected arrives with no new trigger. The guest and the wire format do not change.

The personal key and the authorized_keys lines leave together with the relay at L3, for the device book: the device keys of the owner's known Macs. A key only in `authorized_keys` then never reaches a guest.

## Consequences

- `ssh <workspace>` works on a Mac that has no `~/.ssh` at all.
- Whoever may log into a Mac may still reach its VMs until L3; the sentence of `docs/reference/machines.md` says so.
- L3 rewrites `guest-trusts-device-key-first.spec.yaml` on purpose, and writes `guest-never-trusts-a-foreign-key.spec.yaml` red first.
- Open: the real Secure Enclave signature accepted by a real guest sshd is not proven by the suites (`vm-sshd` accepts any key); it takes one acceptance run on a fresh macOS user.
- Open: whether the image key file is owner-readable on a real install (`stat -f %Sp ~/.local/state/station/images/key` on development-macos), on which the copy's equivalence rests.
