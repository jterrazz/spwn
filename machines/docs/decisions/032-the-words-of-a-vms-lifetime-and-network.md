# ADR-032: The words of a VM's lifetime and network

**Status:** Accepted
**Date:** 2026-11-11

Amends [ADR-017](017-servers-and-workspaces.md): its kind `mac` and its lifetime words Throwaway and `--permanent`. The Machinefile's `vm:` block ([ADR-030](030-init-makes-a-machine-known-apply-carries-its-machinefile.md), [ADR-031](031-the-catalog-defines-the-machinefile-chooses.md)) is written in these words, so the code, the CLI, the JSON and the pages take them before the Machinefile lands.

## Context

A Machinefile says what a machine is with three keys, `kind`, `lifetime` and `network`, and the words of the code did not match them: the kind of a physical machine was `mac`, which names a brand where the system is already its own word; the lifetimes were `throwaway` and `permanent`, while the glossary rejected `ephemeral` and `persistent`, the words the Machinefile's design chose; and the network was a flag whose default had no name. Writing the Machinefile in one set of words and the product in another would have kept two vocabularies for one concept.

## Decision

- **Kind:** `computer`, `workspace` or `server`. A computer is a physical machine the owner owns; a Mac is a computer whose system is macOS, and every sentence still says "Mac" and "This Mac" for it; a computer whose system is another is called a computer (`machine.Kind.Word`).
- **Lifetime:** `ephemeral` (deleted when it powers off from inside; the default) or `persistent` (kept until deleted); a server's is always on. The flag is `--persistent`.
- **Network:** `nat` (behind its Mac, no address on the LAN; the default) or `bridged` (its own address on the LAN; macOS images). The values are lower case; the word in capitals stays banned in pages, and sentences say "behind its Mac". They are the hypervisors' words, not Docker's, whose `bridge` means the opposite.

The old words are read for one release and written only to a peer that reads no other (below):

- `mac` as a kind, in a store or on the wire between two Macs of different versions (`machine.Kind` decodes it as `computer`);
- `throwaway`, `permanent`, and v2's `temporary` and `lasting`, as a lifetime, in a record, a request or a call between two Macs;
- `--permanent`, a hidden alias of `--persistent` that names the new words on stderr.

Between two Macs of different versions, each is written the words it reads, for one release. The boundary is one constant, `remote/api.WordsSince` (v3.1.0, the release of these words), and one function, `remote/api.OlderWords`, writes `mac`, `throwaway` and `permanent` for the new words in the `kind` and `lifetime` fields of a frame:

- a request goes in the older words to a far side whose `Hello` announces a version before the boundary;
- an answer goes in the older words to a caller whose hello names such a version, or none, as every older station's hello does; a caller now names its own version in its hello;
- `machines create` sends no lifetime for the default, which every station's create reads as its own default, so only `--persistent` needs the older word.

A build between two releases counts as the release it follows, so it may be sent the older words, which it still reads; a peer that runs this very build, or names no release (`dev`), is written today's.

The machine store's record moves to version 2: its Migrator, run by `station init`, rewrites a record that wrote the old lifetime words, and an older station refuses a record at version 2 as a newer one.

`station.status/v1` keeps its words byte for byte ([ADR-004](004-station-status-json-is-a-contract.md)); it leaves with v2's other words.

## Consequences

- Every `--json` document that names a kind or a lifetime says `computer`, `ephemeral` and `persistent`; a script that compared with `mac` or `permanent` changes once.
- `machines create --persistent --on` a Mac of an older station succeeds: it is sent `permanent`. An older Mac's `station --on` a newer one shows it as a Mac, since the overview's head comes back as `mac`.
- The gate leaves with the aliases: the minor that removes them deletes `WordsSince` and `OlderWords`.
- The retired words join the banned list in the minor that removes the aliases, not before.
- The catalog's placement words (`mac`, `linux`, `macos`, the pin of the specs) stay until the catalog's kinds merge into kind and system; they are not the words of this decision.
