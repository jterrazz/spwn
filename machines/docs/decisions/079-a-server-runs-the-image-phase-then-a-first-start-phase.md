# ADR-079: A server runs the image phase, then a first-start phase

**Status:** Proposed
**Date:** 2026-11-23

**Amends:** [ADR-044](044-an-image-is-baked-from-its-machinefile.md): its line "A server's Machinefile is refused: station sets nothing up in a server".

## Context

ADR-044 bakes an image from its Machinefile in an image phase, then runs a boot phase at a workspace's first start, and refuses a server's Machinefile. [ADR-078](078-a-server-is-set-up-from-its-machinefile-on-the-owners-verb.md) now has station set a server up from its Machinefile, so that refusal goes, and a server needs the same two phases.

The plan for the studio-infrastructure server (2026-11-20) splits what its Machinefile carries in two:

- **What can be baked**: the base apt packages, sshd's hardening, the CI's public key, the sysctls and the modprobe blacklist, systemd-resolved.
- **What can only happen on the machine itself**: the hostname, the machine-id, growing the root partition, the fstab mounts, one run of `unattended-upgrade` (its spec fails on a fresh machine until it has run once), and the resolver's stub link.

The guest already has a first-start script for the per-machine part (`apps/engine/catalog/images/linux-firstboot.sh`). The rehearsal measured 20 to 24 s of machine preparation on the stand-in, and the host layer at 46 s cold.

ADR-044's boot phase never fails a workspace: what fails is said on the create's page, and the workspace is kept.

## Decision

1. **A server's image is built like a workspace's.** `station images build <machinefile>` runs `station apply --phase image` in the build VM, with the bounds of ADR-044. The image is named after its Machinefile and records the Machinefile, the hashes and the lock.
2. **A server's first start runs, in this order:**
    1. `linux-firstboot.sh` (hostname, machine-id, growpart);
    2. the fstab lines of `vm.mounts` and of the data disk ([ADR-082](082-a-server-keeps-its-data-on-a-disk-of-its-own.md));
    3. `station apply --phase boot`, run as the machine's user with the chain and lock the Mac reads now;
    4. the `tailscale` grant, when `accounts.grants` names it ([ADR-081](081-a-server-takes-one-grant-the-tailscale-auth-key-delivered-once.md));
    5. `run.boot`.

    This is ADR-044's boot phase, with the one grant a server takes.

3. **The first start reads the image's drift once**, off the lock the image recorded, as for a workspace. It reports and never acts: the drift job still skips a server (ADR-078).
4. **A failed first-start phase keeps the server and says so.** It is said on the create's page and in the journal, as for a workspace. A server is not deleted for a failed phase: it may already hold data on its data disk. Its health check reports what does not answer.
5. **`station upgrade --on <server>` runs a bare apply** of the server's recorded Machinefile, then `run.upgrade`. It never runs the image phase again: an item baked later is carried by the bare apply, which carries every item (ADR-044). A change that needs a new image is a rebuild of the boot disk (ADR-082).

## Alternatives refused

- **Bake everything, and leave no first-start phase.** The hostname, the machine-id and the partition size are per machine, and one image serves every server of its Machinefile. `unattended-upgrade`'s first run belongs to the machine's first start, not to the build.
- **Run everything at first start, with nothing baked.** Every create would then pay the apt installs, and a server would depend on the network at its first start. Its creation would get slower and break more easily, with no gain.
- **A phase of its own for servers.** The image and boot phases already say what runs when. A third phase would split the Machinefile's `phase` key between two kinds of machine.

## Consequences

- S2 lifts the refusal of ADR-044 at the four lines `TODO.md` names for it. A tier-2 `create-server-from` case proves the server is sized, mounted and its check passing, that the mount is back after a restart, and that `upgrade --on` applies one change.
- The same `linux-firstboot.sh` serves both kinds of machine. A change to it is a change to every Linux VM's first start.
- What a server's first start did is in the journal, so the owner can read a broken phase after the fact.
