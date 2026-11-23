# ADR-081: A server takes one grant, the Tailscale auth key, delivered once

**Status:** Proposed
**Date:** 2026-11-23

**Amends:** [ADR-049](049-a-vm-signs-itself-in-the-vault-grants-the-rest.md): a server, which takes no grant under it, takes the `tailscale` grant.

## Context

ADR-049 answers `Action: none` for a server on every account row, and the code refuses a grant to a server (`apps/engine/accounts/adapter/keychain/grants.go:176-177`, `GrantToServer`). The studio-infrastructure server has to join the tailnet as `studio-infrastructure`: the CI reaches it there, on 22 and 6443, and the guard accepts `tailscale0` (`TODO.md`, S5). [ADR-078](078-a-server-is-set-up-from-its-machinefile-on-the-owners-verb.md) gives the join to the Machinefile, not to the CI, so the auth key has to reach the server from the Mac's vault.

The owner's answer of 2026-11-21 (the plan's question 3, recommendation accepted) fixes the kind of key. It is tagged `tag:cluster`, pre-approved, one-off, not ephemeral, and expires in a day. A tagged node has no key expiry, so the re-auth unit studio-infrastructure runs today goes, and a rebuild means minting a new key.

The rest of ADR-049 stands: Touch ID at each grant, the value over stdin and never in argv (ADR-009), never in a log or an answer (ADR-048), and one vault per Mac.

## Decision

1. **`tailscale` is a delivery of the catalog's accounts**: the vault entry's value is sent over stdin to `sudo tailscale up --auth-key="$(cat)" --hostname=<vm>` in the guest, through the engine's guest exec. It is the one grant a server takes. `gh`, `npm`, `claude` and `wrangler` stay refused on a server, with today's refusal.
2. **It is delivered once.** It is given at the server's first start when its Machinefile's `accounts.grants` names it, or by `station grants add tailscale --to <server>` on a running server. The value is spent by its use: the server keeps its node identity, not the key. A delivery that fails (an expired or used key) is said with Tailscale's own reason, and the owner mints a new key into the vault.
3. **A delivered grant is a fact of the journal, not an active grant.** It is never retried, never re-delivered at a restart, and never in `failing`. Removing it from the vault does not log the server out of the tailnet. The device is the Tailscale admin's.
4. **Touch ID is asked for it**, as for every grant.
5. **The accounts row of a server offers `tailscale` alone.** Every other row keeps `Action: none` and its engine line.

## Alternatives refused

- **The CI joins the tailnet for the server**, with the OAuth pair it holds today. The vision puts no infrastructure secret in the VM and gives the join to the machine. The CI would also own the link it reaches the server through.
- **An OAuth client in the server, with a re-auth unit.** That is today's design, and the tagged key makes it useless. It would keep a long-lived secret on the server.
- **A grant kept active and delivered again at each start.** A one-off key cannot be used twice, so every restart would be a failure.
- **Lift `GrantToServer` for every program.** A server runs production. A personal `gh` or `claude` token on it reaches far beyond what its tooling needs.

## Consequences

- S5 adds the Debian method of `remote-access/tailscale.yaml` and the `tailscale` delivery, and lifts `GrantToServer` for that delivery alone, in the keychain adapter and the fake. Its stories cover the grant to a server, the refusal kept for the four other programs, and the value absent from every log.
- A rebuild of the server's boot disk loses its node identity, so it needs a new key. The data disk ([ADR-082](082-a-server-keeps-its-data-on-a-disk-of-its-own.md)) does not keep `/var/lib/tailscale`.
- The spent key stays in the vault until the owner removes it or replaces it with the next one.
- `docs/reference/accounts.md` says a server takes the `tailscale` grant, in S5.
