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
2. **What the apply did not finish leaves the workspace made.** The create exits 0. Its page counts what was installed and set on the boot row (`2 apps installed · 2 settings set`), says one line per package left, `◐ <app>  not installed: <why>`, then `◐ <n> apps to finish  station machinefile apply --on <vm>`. A package that would ask the owner something (a `needs: [terminal]` one, as `github-login`) is one such line, `asks a question: finish it in a terminal`, and the others are carried out all the same: the create asks the apply to leave what asks (`LeaveAsking`, an input of `machinefile.apply`), while `station machinefile apply` off a terminal still refuses the whole run and changes nothing. What requires a package left is not run, and named. An apply that answered for no package (its guest gone, the create stopped, its Mac's wait spent) says why on one line, `◐ <machinefile>  not applied: <why>`, and never fails the workspace; so does an apply the failed mounts or copy kept from running. The boot phase's `Failed` keeps the mount, the copy and the boot phase's own failures.
3. **A server's first start is the same first start.** ADR-079's decision 2.3 is the boot phase this record extends, so a server made from its Machinefile applies it whole too. No server reaches it yet: `create --from` refuses a server's Machinefile, as `machines.md` says.
4. **The apply is bounded in the guest, and kept.** The create hands the guest's apply the packages' share of a phase's bound, 15 minutes (`Within`, an input of `machinefile.apply`): the guest reads it on its clock before each package, and no package starts past it. A package manager killed mid-transaction leaves a half-configured system, so the package running at the bound, installing or waiting on its manager's lock, is let finish within a cap of 3 minutes (`catalog.InFlightCap`, its context's deadline); past the cap its install's process group is told to stop (SIGTERM, relayed by sudo), killed 5 s later, its pipes closed, and the next apt run configures what dpkg was left with half done (`dpkg --configure -a`). Those after it are not run, the machine's run lock is let go once the run returns, and the apply answers with what it did: each package left unstarted says `the apply ran out of time`, the one ended at the cap `still installing when the time ran out`, and that apply is not recorded as one run to its end. The Mac waits the cap and a minute's grace beyond the bound; past that, the page says `not applied: it ran out of time`, never the context's own words. A create the owner stops mid-apply says `the create was stopped before it ended`; the guest's apply then runs on to its own bound, since a call's connection sends no cancel for it (`TODO.md`). The create's journal keeps the apply as its own step, `apply`, with its duration, after `machinefile`; the apply's own journal line is written in the VM, as for any apply.
5. **An apply never installs a kernel into a running machine.** A package of the image phase (`phase: image` in its catalog file, `linux-kernel` the one there is) is installed by the image phase alone, whatever bake list a child Machinefile says, since only `images build` purges the kernels the image keeps no longer and builds the kept one's initramfs. The create's apply and `station machinefile apply` leave it, the create's exit unchanged: an image that lacks it names it on the create's drift row, `◐ Drift  its image is behind its Machinefile  linux-kernel is not baked into the image`, which `station images build` mends. Decided by the orchestrator on the owner's delegation, 2026-11-26.

## Alternatives refused

- **Leave the apply to the owner, said on the create's page.** It is what the code did, and what the owner refused.
- **Run the apply as a shell line in the guest**, as the boot phase is. The call answers with a typed result, one entry per package, so the page counts and names them without parsing text, and it is proved in the fake world, where the call runs the real apply in the VM's slice.

## Consequences

- A create needs the network for every app its image did not bake. With none, the workspace is made all the same, each such app named as unfinished, with the command that finishes it.
- A workspace is usable later: the create's time is the boot phase's plus the apply's. In the fake world the apply takes no time on the world's clock (`from-machinefile-kept-with-its-phases`), so the fake measures nothing; on a real Mac it is HYPOTHESIS (2026-11-26) about 4 s for the create's first steps plus the apply's installs, settled by timing `station machines create --from` of a Machinefile listing `ohmyzsh` and `node` on the tier-2 host.
- An app a Machinefile lists and its image does not hold is installed in the running VM, but a package of the image phase. A child whose `vm.image.bake` replaces `base-ubuntu`'s list without `linux-kernel` still bakes it, so its image boots that kernel; an image built before it was of the image phase lacks it, and its create installs no kernel and names it on the drift row (`boot-phase-old-image-shows-drift`, `from-machinefile-installs-no-kernel`).
- The later lots: L2 deletes the boot phase, so `run.boot` becomes `run.create`, run after the apps, and the create's step is called **Set up**; L3 gives the New machine Guide its Set up rung, the apply's progress as its live line and the repair of what is unfinished as a call.
