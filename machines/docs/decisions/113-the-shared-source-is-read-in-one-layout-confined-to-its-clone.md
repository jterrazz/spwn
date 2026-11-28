# ADR-113: The shared source is read in one layout, confined to its clone

**Status:** Proposed
**Date:** 2026-11-28

**Implements:** [ADR-109](109-one-repository-declares-every-machine.md) §1 (the tree, kits, names) and §5 (`run.upgrade` on a computer), where it leaves the reader's choices open. Each choice below was taken by the reader lot on the orchestrator's delegation, the smallest design where ADR-109 is silent, and waits for the owner's acceptance.

## Context

ADR-109 decides the repository's tree and that the shared source is read in it, with the same clone, the same forward-only pull and the same refusals. It does not say how the old `catalog/` and `machinefiles/` layout ends, where in a repository the tree may sit, which machine's folder an engine reads, how far a read may go, or where the `run.upgrade` refusal stands.

## Decision

- **One layout, no detection.** The reader reads `machines/<computer>/`, `kits/` and `packages/<category>/`, under the source's root; a repository with none of them gives no file. The old `catalog/` and `machinefiles/` folders of a shared source are not read, with no alias and no second mode. The owner's own folders keep their layout.
- **`source.path` is a preference.** The source's optional `path` names the folder of the repository the tree is under. It must be a valid relative path with no `..` and not `.`; anything else is refused before a file is read, `machinefile.outside-the-source`.
- **Every link is refused**, under the layout and on the way to it, even one that points inside the clone: a link is never followed in a remote's content, `machinefile.outside-the-source` naming the link.
- **Bounds before reads.** An entry is judged by its stat before it is opened: a file past 256 KiB is refused, and a layout past 1024 files. Anything not a regular file is refused with its path.
- **The machine's own name selects its folder.** The engine reads `machines/<its profile's name>/` alone; a VM's engine reads a folder named after itself, which holds none, so a VM resolves through its template, as today.
- **Logical names stay.** A read file keeps its repository path for every origin a user sees (`studio@1a2b3c4/kits/essentials.kit.yaml`), and joins the library under the name its folder derives: the computer's file is named after the computer, a VM file and a kit after their file. Two files of one name in one library are refused, naming both.
- **The folder checks the file.** A kit that says `machine` or `vm`, a `host.machine.yaml` whose `machine` is not a computer, and a VM file whose `machine` is missing or a computer are refused at their line.
- **`run.upgrade` is refused at resolve time**, for any file of the shared source a computer's chain reads (its own file, a kit, a kit's kit), `machinefile.upgrade-on-a-computer`, naming the file. The owner's own file on that computer may say it: ADR-109 §5 guards the push rights to the shared repository, and the owner's folder is not reached by a push.
- **The computer is the profile's, not the files'.** The refusal above fires when this machine's profile says a computer as much as when the chain does, and a computer refuses to apply or upgrade a Machinefile whose chain says a workspace or a server, `machinefile.another-kind`: a computer bound to `machines/<computer>/<vm>.machine.yaml` would otherwise run that file's `run.upgrade` at night. The reads part from the acts at the machine they resolve against: `plan`, `drift` and `explain` read such a file as its VM would, since the window draws each Machinefile's plan on This Mac, and `apply`, `update` and the night resolve against the profile's kind. So a computer acts on one shared Machinefile, its own `host.machine.yaml`; a computer's Machinefile of the owner's own may bear any name.
- **`files/` is a seam.** It is not read here: the lot that reads a `file` setting's `from` reads it.

## What a push can still run on a bound computer

The refusal of `run.upgrade` takes away one line of the shared source, not every command it can bring to a computer: a package of `packages/`, or a kit that lists one, still runs its own commands on the computer each night and at each `update`. Each path below runs what a push to the repository wrote, with the computer owner's rights:

- **A script method's update and re-install.** A package's `upgrade` runs in `bash -c` at each update (`apps/engine/apps/adapter/script/script.go:97`), and a script with none is installed again, running its `run` or its fetched `url` (`script.go:92`).
- **The re-install of a locked app that is absent.** An app the lock holds and the machine no longer has is installed anew by its method at the night's upgrade (`apps/engine/machinefile/calls/upgrade.go:105-108`, `upgrade.go:216`), so a push that changes that package's method changes what runs.
- **`check:` lines.** A script's or a shell line's `check` runs in `bash -c` each time the app is read (`apps/engine/apps/calls/inventory.go:366`, run at `inventory.go:395`), the plan's reads included.
- **Shell methods.** A shell line is its install and its update, in `bash -c` (`apps/engine/apps/adapter/shell/shell.go:34-37`, `shell.go:45`, `shell.go:58`).
- **`then:` hooks.** The settings a package names in `then` are applied after it is installed or updated (`apps/engine/apps/calls/operations.go:92`, `operations.go:468`), and a setting runs its own method.

What stands against such a push today:

- **The forward-only clone.** A pull takes only a commit that descends from the clone's own (`apps/engine/machinefile/contract/source.go:19-27`), so a rewritten history or a force-push is refused and the clone stays where it stood; a push on top of the history is taken.
- **A protected branch.** The repository's host can refuse a push to its default branch without review; that rule is the host's, not Machines', and Machines does not read whether it is on.

What does not exist:

- **No signature.** A pulled commit is read whoever signed it, or none did.
- **No review gate.** The night pulls (`apps/engine/machinefile/calls/upgrade.go:193`) and runs what it pulled in the same pass; no commit waits for the owner's word before a computer runs it.

So the push rights to the shared repository are still the trust boundary of every computer bound to it, as ADR-109 §5 says; this decision narrows what a push reaches, it does not close it.

## Consequences

- An owner's shared source in the old layout reads empty until it is moved into the tree; no machine of the owner's reads one today (ADR-109 §1).
- The bare name of a VM two computers hold (ADR-109 §2) and a VM bearing its own file's name (ADR-109 §3) are not decided here: they are the router's and the create's.
- A repository may carry the tree under any folder, so the Agents' and the OS's folders can sit beside it.
