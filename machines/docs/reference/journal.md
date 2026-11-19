# The journal

Each Mac keeps thirty days of what was done on it and on its VMs, and `station journal` reads every Mac's lines by day. This page holds where a line is kept, what it says, how a read groups and pages it, and what a read refuses.

## Where a line is kept

A line lives on the Mac that holds the machine it concerns: the Mac itself, or the Mac of a VM ([ADR-061](../decisions/061-the-history-is-kept-by-the-mac-that-holds-the-machine.md), a proposal). So a deleted VM's history stays with its Mac, and a stopped VM's reads like any other.

The file is `{state}/station/journal.jsonl` ([Footprint](footprint.md)). Its first line names its schema, `{"schema":"station.journal/v1"}`, and every other line is one entry. It is written under its `.lock`, through a sibling file and a rename, and a line older than thirty days is dropped at the next write.

A VM keeps no journal: inside a server or a workspace both calls are refused as not for this machine.

## What a line says

| Field     | Holds                                                                                                          |
| --------- | -------------------------------------------------------------------------------------------------------------- |
| `At`      | the instant it was done, on the Mac's clock, RFC 3339                                                          |
| `Machine` | the machine it concerns: the Mac or one of its VMs                                                             |
| `Call`    | the call that made it; empty for an event the engine tells                                                     |
| `Text`    | what was done, in the engine's words (`gate-2 deleted`, `node`)                                                |
| `Detail`  | the sentence on it (`8 GB freed`, `download timed out`), its secrets withheld                                  |
| `Subject` | what it links to: `Kind` (`app`, `machine`, `setting`, `skill`, `image`) and `Name`; absent when nothing links |
| `By`      | `you`, a session's name, `the night` or `station`                                                              |
| `Purpose` | a create's purpose, as it was asked                                                                            |
| `Outcome` | `done`, `failed` or `held`                                                                                     |
| `Upgrade` | `From` and `To`, for a line that moved an app                                                                  |
| `Retry`   | the call that does it again (`Call`, `Args`, `On`), on a failed line that may be                               |
| `Night`   | the instant of the night run it belongs to; absent for any other line                                          |

## Reading it

`journal.read` asks each Mac you know for its lines (`journal.kept`, four Macs at a time) and answers them grouped:

- **By day, newest first.** Each group carries its `Day` (`YYYY-MM-DD`) and its `Title`: `Today`, `Yesterday`, else the day in words.
- **A night apart from its day.** A night's lines are their own group, `Last night` for the newest, `Night of <day>` before it, with a `Sum` (`2 moved, 1 failed`). Its lines that read alike on several machines (the same subject, upgrade and outcome) are one line whose `On` names each; a failed line stays per machine, so its `Retry` names it.
- **Narrowed.** `Mac` reads that Mac and its VMs, `Machine` one machine, `Subject` one app's or machine's lines. A narrowed read carries no `Sum`.
- **The newest night alone.** `Night: true` answers only the newest night group of the Macs asked.
- **Gone.** A subject whose thing no longer exists at the read (a machine no Mac lists, a package the catalog no longer knows) keeps its name and carries `Gone: true`. A gone machine's lines are read by `Subject`, since `Machine` names a machine some Mac still holds.

A read covers `Days` days (7 when none, 30 at most) before `Before`, which is excluded; no `Before` reads up to today. `Next` is the `Before` of the next page: the day after the newest older day that holds a line, empty when none is kept. So paging by `Next` skips no day that has a line.

A Mac that does not answer, or whose station keeps no journal yet, is an `Unheard` row (`Mac`, `Why`), never a refusal: the other Macs' lines still read. A read with no group and nothing unheard carries the engine's sentence in `Empty`.

## Refusals

| Code                      | When                                                      |
| ------------------------- | --------------------------------------------------------- |
| `journal.unknown-mac`     | `Mac` names no Mac you know                               |
| `journal.unknown-machine` | `Machine` names no machine a Mac holds                    |
| `journal.bad-day`         | `Before` is not a `YYYY-MM-DD` day                        |
| `journal.unreadable`      | This Mac's journal does not read; its subject is the file |

## What writes it

A gesture declares the lines it keeps (`journal: true` in [calls.json](api.md#callsjson)), and the Router of the Mac that holds its machine keeps them once the call ends ([ADR-063](../decisions/063-a-gesture-is-kept-by-the-router-of-its-mac.md), a proposal):

- **Where.** A call this Mac runs, for itself, from the CLI, a frame or its engine's socket, is kept here; so is a call it hands to one of its VMs. A call it hands to another Mac is kept by that Mac's own server, and a VM keeps none.
- **What.** A gesture that ran is kept, `done`, or `failed` with its error worded as the CLI words it when it ended in a failure or was interrupted. One refused before any effect (refused, asked for a confirmation, not for this machine, declined) keeps no line. A start or a stop that found the VM as asked keeps none, and so does a gesture run inside another (the `apps.update` of a `machinefile.upgrade`): the outer one's lines say what was done.
- **Who.** `By` is the session `STATION_SESSION` names in the CLI's environment, or the frame's `by`, else `you`; a create's line is its creator's. The nightly upgrade's lines are `the night`'s, each carrying its night's instant in `Night`. A name that is no session's is refused before the gesture runs (`call.bad-input`, `by`). A far binary from before `by` reads a session's call as yours.

The gestures that keep lines today:

| Call                                                                     | Line                                                                                                         |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ |
| `machines.create-workspace`, `machines.create-server`                    | `<vm> created`, with its purpose and its creator                                                             |
| `machines.delete`, `machines.restart`, `machines.start`, `machines.stop` | `<machine> deleted` (`restarted`, `started`, `stopped`), `<machine> not deleted` when it failed              |
| `disk.free`, `disk.free-unused`                                          | `<row> freed` per row it freed, with its sentence; `<row> not freed` and its reason for a row that broke     |
| `apps.update`                                                            | one per app it updated or failed to, a failed one with `apps.update` as its `Retry`                          |
| `machinefile.apply`                                                      | `<file> applied`, what it installed and removed counted                                                      |
| `machinefile.upgrade`                                                    | one per app it moved (`Upgrade`), left waiting (`held`) or failed to move; `<file> upgraded` when none moved |

## Related

- [The API](api.md): `journal.kept` and `journal.read` on the wire
- [Commands](commands.md): `station journal`
- [Glossary](../concepts/glossary.md): journal, line
