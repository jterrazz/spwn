# ADR-078: A server is set up from its Machinefile, on the owner's verb alone

**Status:** Accepted
**Date:** 2026-11-23

Accepted by the owner on 2026-11-23, on the owner's word, with the answers recorded below.

**Amends:** [ADR-017](017-servers-and-workspaces.md): station now sets a server up, from its Machinefile, at its creation and on `upgrade --on`.

## Context

ADR-017 says station sets up nothing inside a server: the server's own tooling (Ansible) owns its inside, and station keeps it alive and checks its health. The code holds that line in four places: `machines create --server --from` is refused (`apps/engine/surface/cli/machines/create/command.go:191-192`), so is a server's Machinefile in the create call (`apps/engine/machines/calls/create.go:82-83`), with the words "Machines sets up nothing inside a server" (`apps/engine/machines/template.go:76-78`), and a grant to a server is refused (`apps/engine/accounts/adapter/keychain/grants.go:176-177`).

The owner accepted on 2026-11-20 a vision for studio-infrastructure that moves that line, and its five recommendations on 2026-11-21 (`TODO.md`, "studio-infrastructure on a station server: the spwn lots"). The cluster's machine is a Debian 13 server that station builds from a Machinefile on a Mac mini. studio-infrastructure's CI is the only deployer of what runs on it, over Tailscale and ssh, and it never touches sshd, tailscaled or the resolver. The Machinefile owns the rest of the machine: the base apt, UTC, unattended-upgrades, sshd's hardening, the CI's public key, systemd-resolved and its upstreams, the sysctls, the modprobe blacklist, and Tailscale.

The rehearsal of 2026-11-20 (a Debian 13 arm64 stand-in on development-linux, fake secrets) showed why the split matters. k3s pins `resolv-conf=/run/systemd/resolve/resolv.conf`, and without systemd-resolved no pod sandbox started, so the cluster's deploy hung 515 s. What the machine carries has to be there before the deployer runs, and only the machine's builder can promise that.

A server runs production. A workspace is changed by its Mac without asking: every night at 03:00 (`docs/reference/machinefiles.md`, "Every night") and by the drift job after each boot. The drift job skips a server today (`apps/engine/engine/process/drift.go:21-25,34`).

## Decision

1. **A server is made from its Machinefile.** `machines create --server --from <machinefile>` builds the server from the image named after it and records the Machinefile on the server, as a workspace's create does (ADR-044). How it runs its phases is [ADR-079](079-a-server-runs-the-image-phase-then-a-first-start-phase.md). What a server's Machinefile may say is [ADR-080](080-kind-server-is-a-usable-machinefile.md).
2. **Station sets a server up at two moments only**: its creation, and the owner's `station upgrade --on <server>`, which applies the server's recorded Machinefile inside it. The plan is shown before anything runs, as for every apply.
3. **Nothing else changes a server.** The night upgrade, the drift job and any other engine job never act on a server. The drift job keeps skipping it: a server is read against its Machinefile only when the owner plans or upgrades it.
4. **A gesture that changes one item stays refused on a server.** Installing an app, applying a setting or granting an account one at a time is refused, as ADR-017 said. On a server the Machinefile is the only route, so what the server carries can always be read back from one file.
5. **Accounts stay refused on a server, except the `tailscale` grant**, which [ADR-081](081-a-server-takes-one-grant-the-tailscale-auth-key-delivered-once.md) decides.
6. **What ADR-017 said of a server's life stands.** It is always on, starts with its Mac's engine, boots again after a crash with a wait that doubles from 5 s, and is checked by its four health steps. A failing check never restarts it. Stopping or deleting it asks its name typed.

## Alternatives refused

- **Keep ADR-017 and let Ansible set the machine up.** That is the state the vision leaves. The CI would own sshd, tailscaled and the resolver, which it reaches through. A wrong sshd change from CI cuts CI's own access.
- **Treat a server as a workspace with a long life.** It would take the night upgrade and the drift job. A production machine would then change at 03:00 with no one there, and a version range moved by an upstream release would reach the cluster unseen.
- **Allow per-item gestures on a server.** A one-off setting would make the machine differ from its file. Then the next `upgrade --on` would undo it or report drift, and the file would no longer say what the server is.

## Consequences

- S2 lifts the four refusals named in the context. Each refusal story turns into a story of a server made, and the error `ServerTemplateError` goes.
- A change to the server's machine is a change to its Machinefile, merged in studio-infrastructure, then `upgrade --on` run by the owner on the Mac that hosts the server. The CI never creates, upgrades or deletes the server.
- A server's Machinefile can change while no one has applied it. Until the owner runs `upgrade --on`, the server stays as it was set up, and only a plan shows the difference.
- A server's window shows its Apps and Settings sections read-only: the owner reads what the Machinefile gave it and changes nothing there, by the owner's answer of 2026-11-23. The window lot decides how, and the other sections ADR-017 refuses (Disk, Accounts, Background and Awake) stay as that record says.
