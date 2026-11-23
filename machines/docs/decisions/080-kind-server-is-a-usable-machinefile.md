# ADR-080: `kind: server` is a usable Machinefile

**Status:** Accepted
**Date:** 2026-11-23

Accepted by the owner on 2026-11-23, on the owner's word, with the answers recorded below.

**Amends:** [ADR-033](033-a-machine-is-declared-by-its-machinefile.md): a Machinefile whose `machine.kind` is `server` is planned, applied and made into a server.

## Context

ADR-033 lists `server` among a Machinefile's kinds, but a server's Machinefile can only be decoded. The catalog knows the role (`apps/engine/catalog/package.go:72`), and the wire marks a template as a server (`apps/engine/wire/machines_templates.go:44`), but every create from one is refused ([ADR-078](078-a-server-is-set-up-from-its-machinefile-on-the-owners-verb.md), Context). ADR-078 and [ADR-079](079-a-server-runs-the-image-phase-then-a-first-start-phase.md) now have station make and set up a server from its Machinefile, so the format has to say which of its keys a server takes.

Three facts shape the answer:

- **A server is always on** (ADR-017). `vm.lifetime`'s words, `persistent` and `ephemeral`, describe a workspace's choice, not a server's.
- **A server is reached through its Mac's NAT and the tailnet.** studio-infrastructure's guard accepts `tailscale0` and the Mac's NAT gateway only, and no public inbound path is allowed (that repository's ADR-006). A bridged address on the LAN would sit outside that guard.
- **The server's Machinefile lives in studio-infrastructure** (`machinefiles/studio-node.machine.yaml`, lot I0). The layers of ADR-033 are the owner's folder, one shared source and the built-in files, so a create needs a way to name a file that sits in none of them.

## Decision

1. **A `server` Machinefile is planned, applied and made**, like a workspace's. It takes:
    - `vm.resources`, `vm.image` (`from` and `bake`), `vm.mounts`, and the data disk of [ADR-082](082-a-server-keeps-its-data-on-a-disk-of-its-own.md);
    - `apps`, `settings`, `ports` and `run`;
    - in `accounts`, only the `tailscale` grant ([ADR-081](081-a-server-takes-one-grant-the-tailscale-auth-key-delivered-once.md)).
2. **A server's chain refuses three things:**
    - `vm.lifetime`, because a server is always on;
    - `vm.network: bridged`, because a server is reached through its Mac and the tailnet only;
    - `accounts.signins`, and any grant other than `tailscale`, because a sign-in is refused for a server (ADR-049).

    Each refusal names the file of the chain that says it, as ADR-033's plan errors do.

3. **`--from` takes a name or a path.** A name is resolved through the layers, as today. A path names a Machinefile file outside the layers. Its `extends` still resolve through the layers, and its overlay is `<name>.local.yaml` beside it.
4. **A server's record keeps where its Machinefile came from**: the name, or the absolute path of the file. `upgrade --on` reads it from there again. A path that no longer reads is refused, naming the path, and the server is left as it is.

## Alternatives refused

- **Refuse `vm.mounts` on a server, now that it has a data disk.** A server may still want a read-only folder from its Mac. ADR-044's rule that an always-on server starts without a missing folder already covers a server.
- **Copy a path's file into the owner's layer at create.** ADR-033 lets a surface write only the overlay. A copy would become a second owner of the file, and it would go stale when the repository changes.
- **Make studio-infrastructure the shared source.** There is one shared source. It holds the owner's Machinefiles for every machine, not one repository's server.

## Consequences

- The JSON Schema, `words.json` and the CLI goldens change through `make generate` in S2. `machines-server-template.spec.yaml` turns from refused to made, and each refusal above gets its story.
- `machines create --from` and `images build` take a path in S2. `docs/reference/machinefiles.md` and `docs/reference/commands.md` say so in the same lot.
- The Mac that hosts the server holds no checkout, and a path needs the file on its disk. The owner places the file there before a create or an `upgrade --on`, by the owner's answer of 2026-11-23. A download by Spwn is a later decision in its own ADR, remote code under ADR-033's rule.
