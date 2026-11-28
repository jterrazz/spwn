# ADR-109: One repository declares every machine

**Status:** Proposed
**Date:** 2026-11-28

**Supersedes:** the line "Where they live: open. Not a git repository of their own (the owner declined, 2026-11-10)" of `TODO.md`'s "Blueprints".

**Amends:** [ADR-047](047-a-catalog-entry-is-a-package-a-machine-file-is-name-machine-yaml.md) (where a Machinefile lives in a shared source), [ADR-033](033-a-machine-is-declared-by-its-machinefile.md) (how the files of a chain merge).

The owner decided the repository, its tree and its kits on 2026-11-28; the naming rule, the merge rules and the security rule below were decided by the orchestrator on his delegation that day, each marked where it is the owner's to overturn. The record waits for his acceptance.

## Context

The owner wants every machine he runs, the agents and the OS declared in one repository, which he will show people (2026-11-28, verbatim):

```text
tout machine/agent/os sera gérable dans un repo unique et ça sera un argument à montrer aux gens.
```

```text
studio-fleet parfait je pense.
```

```text
3 Machinefile justement oui stp. Et à faire de manière composable. Par use case métier je pense, et avec ensuite mes machines nommées par nom qui extend/mergent ces configs.
```

He validated the tree ("oui parfait") and the kit names ("ok"), and answered where the folders sit, at the repository's root or under a `spwn/` folder as the Agents CLI reads a project today:

```text
1 ta reco, on peut supprimer entierement l'ancien mode
```

And he wants the same development machines on both of his computers:

```text
les machines de dev sont également sur macbook pour profiter de la puissance des 2 ordis combinés sans toucher aux host.
```

Four things of today's product stand against that. A shared source is a fixed layout, `catalog/<category>/<name>.yaml` beside a flat `machinefiles/` (`apps/engine/machinefile/adapter/git/git.go:55`, `contract/folders.go:26-64`). A machine has one name across every Mac, the one `--on`, the `~/.ssh/config` block and `preferences.json` use (`docs/reference/machines.md:21`): a VM's name is held by its record on any Mac, so a second Mac cannot make a VM another Mac's VM already bears (`apps/engine/machines/calls/names.go:24-26`), and a far VM's name is asked of every other Mac and the first that holds it answers (`apps/engine/wire/remote.go:147-177`, `remote/router.go:194-210`). A Machinefile's chain merged by "the last file wins", so two mixins that each said `run.boot` kept only the second's, silently (`apps/engine/machinefile/resolve.go:363-374` before this record). And a Mac bound to a Machinefile upgrades every night unattended from the shared source's newest commit ([Machinefiles](../reference/machinefiles.md), "Every night").

## Decision

### 1. The tree

A fleet is one git repository, named `studio-fleet` for the owner. Its folders sit at its root; the `spwn/` folder the Agents CLI reads a project from today is removed, with no second mode and no alias:

```text
studio-fleet/
├── spwn.yaml                       # the manifest: the project's name, the computers it holds
├── machines/
│   └── <computer>/
│       ├── host.machine.yaml       # the computer named <computer> (kind: computer)
│       └── <vm>.machine.yaml       # a workspace or a server that computer runs
├── kits/<name>.kit.yaml            # a mixin: only ever extended
├── packages/<category>/<name>.yaml # the fleet's catalog, `station.package/v1`
├── files/                          # what a `file` setting's `from` reads
├── agents/                         # the Agents CLI's agents, as `spwn/agents/` held them
├── skills/
├── workflows/                      # what runs with no human (the owner's "factory")
├── wiki/                           # the OS
└── docs/
```

- **A kit is a mixin and nothing else.** A `.kit.yaml` that says `machine:` or `vm:` is refused at its line: a kit composes what a machine carries (`apps`, `settings`, `accounts`, `ports`, `prefer`, `run`, `extends` of other kits), and only a file of `machines/` says what a machine is and how it runs.
- **A machine file says a machine.** `host.machine.yaml` says `kind: computer`; a `<vm>.machine.yaml` says `workspace` or `server`. A computer reads `kits/`, `packages/`, `files/` and its own folder `machines/<computer>/`, never another computer's: the Machinefiles of one library are those, so two computers' files of one name never meet in one resolution.
- **Names.** A file's `name:` equals the name its place derives, and another is refused naming both: `machines/<computer>/host.machine.yaml` is `<computer>`, `machines/<computer>/<vm>.machine.yaml` is `<vm>`, `kits/<name>.kit.yaml` is `<name>`, `packages/<category>/<name>.yaml` is `<name>`. `host` is no VM's name. A kit and a machine file of one name in one computer's library are refused, as two Machinefiles of one name in one source are; a fleet's file of a built-in's name replaces it whole, as any stronger source's does.
- **The fleet is the shared source.** Preferences name it as they name a shared source today, and it is read in this layout: the lot that reads it (the reader lot below) changes the source's reader and its folders, the same clone, the same forward-only pull, the same refusals.

### 2. One development machine on two computers

The owner's ask is `machines/jterrazz-macbook-pro/development-linux.machine.yaml` beside `machines/jterrazz-mac-mini/development-linux.machine.yaml`: one name, two VMs, one on each computer. Three rules were weighed.

- **A. One namespace, as today.** Every VM's name is unique across the owner's Macs; he writes `development-linux-macbook` and `development-linux-mini`, both extending one kit. Against it: it refuses what he asked, and every VM's name grows its computer's.
- **B. Always qualified.** Every VM's name is `<vm>.<computer>`, in tart, ssh and `--on`. Against it: it renames every VM there is, and `--on api` with it, for a long name where one Mac holds the VM, which is nearly always.
- **C. Scoped by its computer, short where it is one.** A VM's name is unique on its computer; its full address is `<vm>.<computer>`; the bare `<vm>` reaches it wherever it names one machine. Against it: a bare name two Macs hold has to be refused, never guessed.

**Rule C is decided.** _The owner's to overturn: one namespace (A) is the smaller change._

- **A VM's name is held on its own computer.** A create refuses a name This Mac's VMs or any Mac bears, and no longer one a VM of another Mac bears.
- **`<vm>.<computer>` is a machine's full address**, everywhere a machine is named: `--on development-linux.jterrazz-mac-mini`, `station use`, `station machines ssh`. The dot is unambiguous: a machine's name is lower-case words joined by dashes, never a dot, and ssh reads `a.b` as one alias where `a@b` would be a login. On the wire nothing changes: the address is `machine.Ref{Name, On}` (`apps/engine/core/machine/machine.go:38`), which already carries the Mac a VM is on.
- **The bare name keeps reaching one machine.** This Mac's own VM of that name first, as today; else the one Mac that holds it; a bare name two other Macs hold is refused before anything is dialed, naming each full address (`development-linux is on jterrazz-mac-mini and jterrazz-macbook-pro: name one, development-linux.jterrazz-mac-mini`), where today the first Mac to answer wins. Every name in use now names one machine, so nothing reached today changes.
- **The ssh blocks follow.** A far VM's block is `Host <vm>.<computer>`, and also `Host <vm>` while no other Mac holds that name.

The code the reader lot touches: the claims a create checks (`apps/engine/machines/calls/names.go`), the router's resolve and the holder lookup, which gather every Mac that holds the name (`remote/router.go:194-210`, `wire/remote.go:147-177`), the parse of `--on` and `use`, the `~/.ssh/config` writer, a refusal code for a bare name that names two machines, `machines.md` ("Macs and VMs", "On another machine", "SSH") and the glossary's machine name.

### 3. A fleet's VM bears its own file's name

Today a VM is never named like a Machinefile its template's chain reads (`apps/engine/machines/template.go:103-108`), since that VM's own changes, `<vm>.local.yaml`, would change the file for every machine made from it. A fleet's `machines/<computer>/<vm>.machine.yaml` is one machine, not a template others are made from, so **the VM it makes bears its name**: its own changes and the file's overlay are one file, which the chain reads once (`chain` already skips the machine's own overlay when the machine names a file of the chain). The refusal stays for every other name of the chain (a kit's, the built-in base's), and a machine file is made into its one VM: `--from` naming it with another VM's name is refused.

### 4. Kits compose

The chain is the same walk: `extends`, in order and depth first, a file read once however many extend it, a cycle refused. What changes is the merge ([Machinefiles](../reference/machinefiles.md), "The resolution"):

- **`run` concatenates.** Each file's `run.<phase>` is its own step, run in the chain's order, the step named by its file in the plan, the run and `explain`; no file's lines replace another's. A step that fails ends the phase there: the steps after it are not run, and say so. No built-in Machinefile and no example of the repository relied on a child replacing its parent's `run` (read 2026-11-28: the built-in `base-ubuntu`, `base-debian` and `base-macos` say none).
- **`accounts` unite, under three rules of control.** `signins` and `grants` are the union over the chain, each program once, in the order the chain first lists it, so kits compose. (a) A file that gives a key as an explicit empty list (`signins: []`) clears what every file before it in the chain gave for that key, in a plain file as in an overlay; the decoder tells `[]` from a key not said, and an overlay still sets each key it says. (b) A later file naming a program under the other key than a file it extends moves it there, the most specific file winning: a base signs in to `wrangler`, its child grants it, and it is granted, with no refusal. (c) The sign-in and grant conflict is judged once, on the final merge, and only between siblings, two files neither extending the other; the refusal names both files, and an overlay or the top file settles it by (a) or (b). Every `explain` line of an account names the file that gave it. _Decided by the orchestrator on the owner's delegation, 2026-11-28._ Migration: a child that narrowed its parent's accounts by restating them (`signins: [gh]` under a parent's `[gh, wrangler]`) now keeps the parent's too; since one key holds one list, a file cannot clear and restate it at once, so the narrowed list goes in the child's overlay (`accounts: { signins: [gh] }`, which replaces the key), or a file between the two clears the key with `signins: []` and the child, extending that file, lists `[gh]`.
- **`ports`, `prefer`, `vm`, `vm.mounts` and `vm.image.bake` are the last file's in the chain's order** that says them, as before: `ports`, `prefer`, `vm.mounts` and `vm.image.bake` whole (an overlay's `ports` key by key), every other key of `vm` by key. The last in the chain's order is not always the most specific: of `top` extending `b` and `c`, both extending `d`, the order is `d`, `b`, `c`, `top`, so when `b` and `c` both say `ports` and `top` says none, `c`'s stand and `b`'s are dropped with no refusal. Siblings are judged only for apps, settings and accounts; for these keys the file extending both says them to settle it.
- **Apps and settings by name.** A file still overrides what the files it extends say of a name. Two files of which neither extends the other (siblings) that give one app or setting different entries are refused, naming both files, each at its line, and the name. Entries compare by meaning, so the same entry twice is one: versions as ranges (`latest` is no version), a manager the package takes by default there as none, and two entries that differ only in `optional` are one, the app required if any file requires it; a built-in base and a kit follow the same rule, with no exception by source (`base-ubuntu`'s optional `git` beside a kit's bare `git` is one entry, required). _Decided by the orchestrator on the owner's delegation, 2026-11-28._ The file that extends both settles it by saying the entry itself. Of a diamond (A extends B and C, both extend D), D is read once, and what B and C inherit from D is no conflict.
- **A chain is bounded**: 8 Machinefiles deep at most, the one resolved counted, and 32 in all, overlays not counted; past either it is refused with the bound, a chain too deep printed along its longest path, one with too many files naming the file past the bound. A cycle is judged first, so one longer than 8 is said as a cycle. The work is bounded too: a file that lists one Machinefile twice in `extends` is refused at its line, and the walk reads each file once, a file reached again deeper judged at that depth, in the resolution as in the rights a VM's Mac reads.
- **A mixin stays extend-only**: one that is planned or applied is refused, and `explain` reads its own chain.

### 5. Push rights are code execution

A Mac bound to a Machinefile of the fleet takes the fleet's newest commit every night at 03:00, unattended, and runs what changed there: a package's `script` method, a `run.upgrade` line. So **whoever can push to the fleet's default branch runs code on every bound Mac**, as its owner, with nobody present.

- The fleet's default branch takes the owner's pushes alone, through a protected branch, and never a force-push (the source moves forward only already, and refuses a rewritten branch).
- **No computer's chain carries `run.upgrade`**: the reader refuses it in `host.machine.yaml` and in any kit a computer's chain reads, so a night on a Mac runs only what the catalog's packages say. A VM's chain may carry it: a VM never upgrades by itself, and its `run.upgrade` runs at the owner's own `station machinefile update`. _The owner's to overturn._ The reader lot builds this guard ([ADR-113](113-the-shared-source-is-read-in-one-layout-confined-to-its-clone.md)): `machinefile.upgrade-on-a-computer` refuses a shared file's `run.upgrade` in a computer's chain, and `machinefile.another-kind` refuses a computer applying or upgrading a VM's or a server's file, the computer being the one its profile says. The packages' own commands still run there; ADR-113 names each path.
- A computer is bound to its fleet file on the owner's word alone: until he says go, the MacBook is read by `plan`, never applied.

## Consequences

- One repository holds every machine, kit, package, agent, skill, workflow and the wiki, and the owner can show it as it is.
- The tree is the Agents CLI's too: its project folders move to the root, `spwn/agents/` becoming `agents/`, and the `spwn/` mode is deleted with the lot that lands it; this record changes no code of the Agents CLI.
- A run step per file changes the JSON documents' `run` from one object to a list, `station.machinefile.plan/v3`, `station.machinefile.apply/v3`, `station.machinefile.upgrade/v3` and `station.machinefile.drift/v4` (the plan's shape, though a drift reads the apply phase, which runs no lines), and the wire's `Applied.Ran` and `Upgraded.Ran` become lists, within the unshipped state revision 2.
- Two kits that disagree on one app are refused rather than one of them winning silently; the owner settles it in the machine's file.
- Host-scoped names cost a refusal where a bare name is ambiguous, and one more ssh block per far VM.
- A push to the fleet's default branch is a deployment to every bound Mac, and is guarded as one.
