# ADR-103: A create applies its whole Machinefile

**Status:** Proposed
**Date:** 2026-11-26

**Realizes:** [ADR-051](051-a-machine-is-a-base-and-its-machinefiles.md) §5: "its first start applies it".

**Amends:** [ADR-044](044-an-image-is-baked-from-its-machinefile.md): its bullets "The boot phase runs at a workspace's first start, and never fails it" and "The owner's own lines run per phase: `run`"; [ADR-079](079-a-server-runs-the-image-phase-then-a-first-start-phase.md): its decision 2.3, the server's `station apply --phase boot`.

The owner decided it on 2026-11-26, asked whether a `create --from` should apply the apps its image did not bake: this record carries his decision and waits for his acceptance of how it is written.

## Context

ADR-051 made a machine a base and its Machinefiles, and said its first start applies them. The code never followed: a create from a Machinefile ran ADR-044's boot phase alone, the packages marked `phase: boot` and `run.boot`, so a Machinefile listing an app its image did not bake made a workspace without it, and the owner had to run `station machinefile apply --on <vm>` himself. Asked whether a create should apply the rest automatically, the owner answered (2026-11-26, verbatim):

```text
2. appliquer automatiquement et je comprends pas cette distinction tout le reste / au démarrage. C'est pas le même script du package ? le gerne de boot je crois qu'on avait créé ou je sais plus quoi.
```

The first words are the decision; the rest questions the split between the boot phase and the apply, which this record leaves to the lots that follow it.

## Decision

1. **A create applies its whole Machinefile.** Once the workspace answers SSH, its Machinefile is copied in and its boot phase runs as ADR-044 says; then, after its grants and sign-ins, the create asks the workspace's engine for `machinefile.apply` of that Machinefile, as a call at the VM through its Mac's engine, never as a shell line. What the image baked reads present and is kept; every other package is installed or set there. An ephemeral workspace is set up as a persistent one; a restart applies nothing.
2. **What the apply did not finish leaves the workspace made.** The create exits 0. Its page counts what was installed and set on the boot row (`2 apps installed · 2 settings set`), says one line per package left, `◐ <app>  not installed: <why>`, a package that would ask the owner something included, since a create has no terminal, then `◐ <n> apps to finish  station machinefile apply --on <vm>`. An apply that answered for no package (its guest gone, its bound spent) says why the same way and never fails the workspace: the boot phase's `Failed` keeps the mount, the copy and the boot phase's own failures.
3. **A server's first start is the same first start.** ADR-079's decision 2.3 is the boot phase this record extends, so a server made from its Machinefile applies it whole too. No server reaches it yet: `create --from` refuses a server's Machinefile, as `machines.md` says.
4. **The apply is bounded and kept.** It runs under the packages' share of a phase's bound, 15 minutes, so a create with no network ends. The create's journal keeps it as its own step, `apply`, with its duration, after `machinefile`; the apply's own journal line is written in the VM, as for any apply.

## Alternatives refused

- **Leave the apply to the owner, said on the create's page.** It is what the code did, and what the owner refused.
- **Run the apply as a shell line in the guest**, as the boot phase is. The call answers with a typed result, one entry per package, so the page counts and names them without parsing text, and it is proved in the fake world, where the call runs the real apply in the VM's slice.

## Consequences

- A create needs the network for every app its image did not bake. With none, the workspace is made all the same, each such app named as unfinished, with the command that finishes it.
- A workspace is usable later: the create's time is the boot phase's plus the apply's. In the fake world the apply takes no time on the world's clock (`from-machinefile-kept-with-its-phases`), so the fake measures nothing; on a real Mac it is HYPOTHESIS (2026-11-26) about 4 s for the create's first steps plus the apply's installs, settled by timing `station machines create --from` of a Machinefile listing `ohmyzsh` and `node` on the tier-2 host.
- The later lots: L2 deletes the boot phase, so `run.boot` becomes `run.create`, run after the apps, and the create's step is called **Set up**; L3 gives the New machine Guide its Set up rung, the apply's progress as its live line and the repair of what is unfinished as a call; L4 lets an app that asks the owner leave only itself unfinished, the others carried.
