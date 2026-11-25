# ADR-085: A Mac run alone keeps a device key that signs with nobody present

**Status:** Accepted
**Date:** 2026-11-23

Amends [ADR-064](064-a-key-never-leaves-its-device.md).

Amended by [ADR-092](092-a-machines-device-book-says-who-may-reach-it.md): the hello sends the line of the key the engine holds; `device-key.pub` is an export only.

Accepted by the owner on 2026-11-23, on the owner's word.

## Context

ADR-064 §1 makes every device key with `[.privateKeyUsage, .userPresence]` under `kSecAttrAccessibleWhenUnlockedThisDeviceOnly` (`apps/engine/platform/darwin/enclave.swift`), so every signature needs an owner present at an unlocked Mac. Its Consequences left open what an unattended Mac does. The new Mac mini answered it on 2026-11-23: with its screen locked, `ssh development-linux-mini` on the mini itself was refused, `agent refused operation`, the agent logging `OSStatus error -25308`. The first ssh after a start had also waited on a password sheet on the mini's screen, which nobody watches. A Mac run alone can never ssh into its own VMs.

A Secure Enclave key's access control is fixed when the key is made and never changes. A Mac that changes how it is run therefore needs another key, and every VM that trusted the old one has to be told.

The repository already says what a Mac run alone is. The Always-on Mac settings group is the owner's opt-in for a Mac that hosts servers (`docs/reference/settings.md`), and the settings judge reads "an always-on Mac is one whose Never sleeps is applied" (`apps/engine/settings/calls/deps.go`). Two other sources were weighed and left:

- `autologin` logs in a fixed user, `jterrazz.agent`, and is not applied on the mini.
- A Mac's own sleep set off in System Settings (`pmset` sleep 0) is a fact any user may set for another reason. It is not the owner's word to Spwn.

## Decision

1. **A Mac run alone is a Mac whose Never sleeps is applied.** That is the `never_sleeps` preference of `preferences.json`, the one the settings judge already reads. No new setting or kind is made.
2. **The device key of a Mac run alone signs with nobody present.** It is made with `[.privateKeyUsage]` alone under `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`, so it signs from the first unlock after the Mac starts, its screen locked or not, and asks no presence. Every other Mac keeps ADR-064's key: an owner present at an unlocked Mac, asked once per engine life.
3. **The file names the key's access.** The present owner's key stays `~/.local/state/station/device-key.enclave`, and the unattended key is `~/.local/state/station/device-key-unattended.enclave`. Each engine start reads Never sleeps once and opens the file of that access. When that file is missing, it makes the key, keeps it, then deletes the other file. So applying Never sleeps, or turning it off, gives the Mac a new key at the engine's next start.
4. **The VMs trust the new key through ADR-077's path.** The engine start writes the new public half to `device-key.pub`. The engine's hello sends it to every guest, and the next walk sends it again when it changed. The guest rewrites its managed block of `authorized_keys` whole, so the old key leaves the VM as the new one arrives.

## Consequences

- The trust trade-off, accepted by the owner: a Mac run alone that is stolen while it runs, or opened by anyone at its session, signs as that Mac with nobody present, and reaches every VM that trusts its key. The key still never leaves the Enclave. A Mac powered off and started again signs nothing until its first unlock, so on a FileVault Mac a thief who cut its power gets nothing.
- A Mac someone uses keeps Touch ID or the password on the first signature of each engine life.
- Applying or undoing Never sleeps changes nothing about the key until the engine restarts. A running engine keeps the key it served at its start.
- The old key is gone once the new one is kept. A far process that pinned the old public half, rather than reading `device-key.pub`, must be told again.
- The darwin tests prove, on a throwaway macOS VM's own Secure Enclave, the protection class and flags each access asks for, and that an unattended key signs with no presence asked. The test of the adapter proves the swap with a fake Enclave. Whether the mini signs with its screen locked is proved by one acceptance run on the mini.
