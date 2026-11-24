# ADR-091: A block on a Mac offers the device key alone

**Status:** Proposed
**Date:** 2026-11-24

Amends [ADR-064](064-a-key-never-leaves-its-device.md) §3 and §4.

## Context

ADR-064 made every managed block on a Mac offer the device key first, then `~/.ssh/id_ed25519`, so that a machine not yet told of the device key still took the file key. The infra audit of 2026-11-24 read `ssh -G` for every station host on the owner's MacBook and found the file key offered BEFORE the device key: the `ssh-key` setting had written `Host *` with `IdentityFile ~/.ssh/id_ed25519` at the top of `~/.ssh/config`, and ssh adds the `IdentityFile` of every `Host` that matches, in the order it reads them. `IdentitiesOnly yes` does not stop it, since that key is named. `IdentityFile none` in a later block does not take it back either (OpenSSH 10.5, measured with `ssh -G` the same day). So any host that accepts the file key was signed into without the presence the device key asks, which the owner approved closing (S1).

The same `Host *` carried `AddKeysToAgent yes` for every host, and test runs whose ssh read the owner's real config (ssh reads the passwd home's `~/.ssh/config`, whatever `$HOME` says) left their throwaway keys in the owner's agent (S3).

## Decision

1. **On a Mac, a block offers the device key alone**: `IdentityAgent` (the engine's agent), the device key's public file, `IdentitiesOnly yes`, and no key file, neither `~/.ssh/id_ed25519` nor the one `machines add --key` named. Off a Mac, where no device key is, a block keeps naming its one key, written under `~`.
2. **The `ssh-key` setting names its key for GitHub alone**: its stanza is `Host github.com` with `AddKeysToAgent yes`, `UseKeychain yes` and `IdentityFile ~/.ssh/id_ed25519`. There is no `Host *` stanza: what the file key is for, now that the device key opens the machines, is GitHub, and a stanza for every host is what put that key in front of the device key and what poured other keys into the agent.

## Consequences

- A machine that does not trust this Mac's device key no longer accepts ssh from it until it is told of the key; ADR-064 §4's bridge, the file key opening every machine until the device book, is closed for the blocks. The relay of the session's agent stays, and serves only what names the engine's socket as `SSH_AUTH_SOCK`.
- The other hosts of the owner's own file still get ssh's default identities, `~/.ssh/id_ed25519` among them, but without `UseKeychain`: a passphrase not already in the agent is asked again there.
- A `~/.ssh/config` that holds the earlier `Host *` stanza keeps offering the file key to every host until that stanza is removed by hand; the setting reads it as no Keychain stanza and adds the GitHub one beside it, never rewriting a line it finds.
- On a Mac, `machines add --key` is recorded and offered nowhere; whether the flag retires is left open.
- Not merged while a machine trusts only the file key: the mini's `authorized_keys` holds `admin@jterrazz.com` alone (2026-11-25), so MacBook to mini ssh would be refused until the owner grants this Mac's device key there or the engine's own door replaces ssh.
