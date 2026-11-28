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
- **`files/` is a seam.** It is not read here: the lot that reads a `file` setting's `from` reads it.

## Consequences

- An owner's shared source in the old layout reads empty until it is moved into the tree; no machine of the owner's reads one today (ADR-109 §1).
- The bare name of a VM two computers hold (ADR-109 §2) and a VM bearing its own file's name (ADR-109 §3) are not decided here: they are the router's and the create's.
- A repository may carry the tree under any folder, so the Agents' and the OS's folders can sit beside it.
