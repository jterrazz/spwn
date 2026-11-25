# ADR-092: A machine's device book says who may reach it

**Status:** Proposed
**Date:** 2026-11-25

An agent leaves the status `Proposed`; only the owner writes `Accepted`.

Amends [ADR-077](077-a-workspace-trusts-its-macs-device-key-first.md), [ADR-064](064-a-key-never-leaves-its-device.md) and [ADR-090](090-a-machine-is-reached-through-its-engines-door-never-through-openssh.md).

Decided by the orchestrator on 2026-11-25, under the owner's delegation of that day, on the open points of the device book's plan: the verbs, the code noun, presence on add, and what the book deletes.

## Context

A VM trusts three things today: its Mac's device key, the Mac's `~/.ssh/id_ed25519.pub`, and every line of the Mac's `~/.ssh/authorized_keys` ([ADR-077](077-a-workspace-trusts-its-macs-device-key-first.md)). So whoever may log into a Mac may reach its VMs, and a key added to that file for any reason opens them all. ADR-077 and [ADR-064](064-a-key-never-leaves-its-device.md) §4 both say the copy and the personal key leave "with the device book", which no record defines. [ADR-090](090-a-machine-is-reached-through-its-engines-door-never-through-openssh.md) builds its door on that book, names its gesture `spwn devices allow`, and lists a `hello` channel.

Three facts bound the design:

- The engine's API marks no frame as come from another machine: a frame relayed by `engine attach` reads like one from the local socket.
- The key agent signs any data it is given, so any process that reaches `agent.sock` can make this Mac's device key sign a file's content.
- [ADR-021](../../../docs/decisions/021-one-program-spwn-one-command-for-everything.md)'s verb table is closed: adding to a set is `add`, and `allow` and `list` exist nowhere.

## Decision

1. **Each machine keeps a device book**: the list of the OTHER devices allowed to reach it. It is `~/.local/state/station/devices.json`, mode 0600, beside `vault.json`. Each entry holds a name, a public key, the channels it may open, when it was added, and a seal. This Mac's own device key is never an entry: its own VMs always trust it. In code the noun is `devices`, never "book", which is the vault's.
2. **An entry passes five rules**, each refused by its own code:
    - its name passes the machine-name rule;
    - its key is `ecdsa-sha2-nistp256` (a device key) or `ssh-ed25519`, and nothing else;
    - no name and no key appears twice;
    - its key is not this Mac's own device key;
    - its channels are a non-empty subset of `spwn-api`, `pipe` and `herdr`, all three by default.
3. **The commands follow ADR-021's table.** `spwn devices` reads the book (`--json`, `spwn.devices/v1`), and works with the engine down and `--on` another machine. `spwn devices add <name> --key '<public key>' [--channels …]` adds an entry. `spwn devices forget <name> --confirm=<name>` takes one away. `add` and `forget` are the engine's calls, each one line of the journal.
4. **`add` asks the owner's presence, every time**, with a reason that names the device and its key's fingerprint, never riding the engine's open key session. Refused or absent, nothing is written. `add` is refused when the request comes from another machine (`devices.add-here-only`): a key is added only by the owner present at the machine that accepts it.
5. **`forget` asks no presence and works from afar.** Taking trust away is the safe direction, and a headless Mac must be able to drop a stolen laptop at once.
6. **A frame from another machine is marked by the transport that carried it.** A server that serves another machine's frames (`engine attach` today) marks every frame it reads as far, and the mark rides every hop after it. A frame may set the mark, never clear it, so a local process that leaves it out gains nothing: it still meets presence.
7. **Each entry is sealed by this Mac's device key.** The seal signs `spwn-device-book/v1`, this Mac's name, the SHA-256 of this Mac's device public key, then the entry's name, key, sorted channels and added time, one per line. Each read checks it against the public key the Enclave's blob gives, never against `device-key.pub`. An entry whose seal fails is ignored, and a finding gives the exact command that adds it again. A book copied from another Mac, an old backup, a hand edit and [ADR-085](085-a-mac-run-alone-keeps-a-device-key-that-signs-with-nobody-present.md)'s key swap each break every seal they touch, on purpose.
8. **The key agent signs logins only.** It signs an SSH user authentication request (RFC 4252 §7) for the key it serves and refuses any other data, so `agent.sock` cannot forge a seal. A connection that binds a forwarded session (`session-bind@openssh.com` with `is_forwarding` set) signs nothing.
9. **A VM trusts its Mac's device key and the book's entries that carry `pipe`, nothing else.** The copy of the Mac's `~/.ssh/authorized_keys` and the `~/.ssh/id_ed25519.pub` line leave. The engine sends the new keys to every running guest as soon as `add` or `forget` ends.

### What the seal is worth

- On a Mac with a present owner, forging a seal takes a Touch ID.
- On a Mac run alone, ADR-085's key signs with nobody present, and any process of the owner on that Mac can load its blob. There the seal guards against accidents only: a sync tool, a restore, a hand edit.

### Amendments

- **ADR-090.** The gesture is `spwn devices add`, not `spwn devices allow`. `hello` is no channel: the first frame of `spwn-api` is already `hello` ([ADR-016](016-api-v2-and-remote-by-api.md)), and ADR-074's budget is measured on it. The door's channels are `spwn-api`, `pipe` and `herdr`, the book's.
- **ADR-077.** Its VM half leaves: the `authorized_keys` copy and the personal key line. Whoever may log into a Mac no longer reaches its VMs.
- **ADR-064 §4.** The `id_ed25519` line leaves guest trust. The relay of the owner's key does NOT leave with the book: until the door, another Mac is still reached through OpenSSH, whose far sshd reads its own `authorized_keys`. The relay leaves with the door's cut.

## Consequences

- After this lands, another Mac's ssh to this Mac's VMs is refused until the owner runs `spwn devices add <that mac> --key …` here, with presence. The empty book's hint says so. No compatibility path is kept.
- The book never writes a Mac's `~/.ssh/authorized_keys`: Mac-to-Mac through OpenSSH is unchanged until the door turns that sshd off.
- A VM block that still offers `~/.ssh/id_ed25519` costs one refused attempt after the device key; the held branch `fix/ssh-device-key-only` removes it.
- On a present-owner Mac, `add` can show two sheets in one gesture: the presence it asks, then the Enclave's own when the engine's key session is not open yet to sign the seal.
- Open: whether the presence sheet of `add` on a Mac with no screen, answered through a remote desktop, is acceptable to the owner.
- Open: the real Enclave signing a seal and a real guest sshd accepting a book key are not proven by the suites, which run the device-key and presence fakes.
