# ADR-064: A key never leaves its device; station is the ssh agent

**Status:** Proposed
**Date:** 2026-11-20

Written 2026-11-20 from the connection architecture's design (option C, "hardened status quo") and its lot L2.

## Context

Every link between the owner's machines that is not a socket of this Mac is OpenSSH: another Mac's engine over `ssh -o BatchMode=yes`, the owner's own `ssh`, editors, `scp` and `git` to a VM through its block's ProxyCommand. Each of them signs with `~/.ssh/id_ed25519`, a private key on disk, made by the `ssh-key` setting, its passphrase in the Keychain. Any process running as the owner can read that file, and one key copied anywhere opens every machine. Replacing OpenSSH with a door of station's own (an mTLS port on the tailnet) was weighed and refused: it trades OpenSSH's audited auth for code station would write, and a shell would then depend on the engine. A permanent Secure Enclave key lives in the data-protection keychain, which refuses a binary without the keychain entitlement: a throwaway macOS VM answered OSStatus -34018 for an ad-hoc-signed and for an unsigned station alike (2026-11-20).

## Decision

1. **Each Mac has one device key, and it never leaves the Mac.** It is a P-256 key made in the Mac's Secure Enclave on the agent's first start and never exported in clear: through CryptoKit's documented API, the Enclave wraps it into an opaque blob that only this Mac's Enclave can use, kept in `~/.local/state/station/device-key.enclave` (mode 0600). No keychain item holds it, so no entitlement is needed, and a copy of the file opens nothing on another machine. ssh sees it as `ecdsa-sha2-nistp256`. Its use asks the owner's presence once, on the first signature, and that answer opens every later one for the life of the agent: Touch ID once per session.
2. **Station is the ssh agent.** `station machines agent` serves the device key on `~/.local/state/station/agent.sock`; it lists, signs, and refuses add, remove and lock. Its own launchd agent, `com.jterrazz.station.agent`, runs it from login on, beside the engine's and not inside it, so an engine restart keeps the shells' agent.
3. **Every managed block offers the device key first.** On a Mac a block names `IdentityAgent` (station's socket), the device key's public file, then the block's own key, and `IdentitiesOnly yes` always, so ssh offers those and nothing else.
4. **Until the machines accept the device key, the agent relays the owner's key.** Every key the agent does not hold is asked of the session's own agent (`SSH_AUTH_SOCK`, never station's own socket), so `~/.ssh/id_ed25519` still opens every machine in between. The relay, and the `id_ed25519` lines of the blocks, leave once the device book authorizes every device key (lot L3).

## Consequences

- The private half of a Mac's ssh identity is in no file: a copied home opens nothing once the `id_ed25519` lines leave.
- Revocation is per device, since no key is shared between two Macs.
- Deleting `device-key.enclave` loses the key: the next start makes another, which every machine must be told of again.
- A machine with no Secure Enclave has no device key: the agent refuses to start there, and its blocks name no agent.
- An unattended Mac has nobody to give presence: what the engine's own ssh does after a reboot is lot L3's decision.
- It supersedes the on-disk default key of the `ssh-key` setting once L3 lands; until then that key stays the one every machine accepts.
