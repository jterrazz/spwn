# ADR-097: A confirmation follows its consequence

**Status:** Proposed
**Date:** 2026-11-25

Reverses the graded table of SPEC §7.5 (the levels by action and kind that `confirm.For` held), and the menu bar's own grading of SPEC A5 (`confirm.Of`'s ephemeral delete asked yes or no).

## Context

The first-user pass of 2026-11-25 found the questions inverted: `machines stop qa` asked "Stop qa?", a stop that loses nothing, while `machines delete linux-1` destroyed a running VM's disk with no word (CLI finding 2). The window asked the name typed to restart a server already down, a repair (GUI finding 8). A "no" printed `Error: qa kept: you said no` (CLI finding 27).

Lot Q2a made a delete of a running VM ask its name, in the call. That opened a second collision: the menu bar graded the same delete by its own facts, asked yes or no, and then sent the VM's name as the call's answer, so the owner's yes stood for a name he never typed.

The owner accepted the rule on 2026-11-25 (question 3 of the QA pass): a running ephemeral VM's delete asks its name typed, a stopped and empty one asks nothing, and a stop asks nothing and says how to start again. His theme: friction follows what can be lost, and the machine never asks what it already knows.

## Decision

What an act can lose decides what it asks, by one table in `core/confirm` that every surface reads: the command line, the window and the menu bar.

| Consequence  | Asks                               | In a script                    | Actions                                                                                                                                                                                                                                                                 |
| ------------ | ---------------------------------- | ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Reversible   | nothing; the outcome says the undo | runs                           | install, share, start, restart a workspace, edit, stop a workspace, stop sharing, undeclare, build an image, relocate, a setting off here, the delete of a stopped ephemeral VM nobody holds, the restart of a server already down                                      |
| Stops others | one yes or no that names them      | `--yes`, or `--confirm=<name>` | stop a server, restart an up server, restart a Mac, stop the engine, let a Mac hosting a server sleep, remove an app or a background task, sign out, remove a password or a grant, overwrite a setting, forget a Mac, delete an image, clean up, free what nothing uses |
| Destroys     | the name typed                     | `--confirm=<name>`             | delete a VM that runs, that someone holds, or that is kept; erase an app's data; delete an agent's Mind; forget a device                                                                                                                                                |

- **One grading.** `confirm.ConsequenceOf(action, kind, facts...)` is the table; `confirm.Of` is its level. The facts are a VM's own (`machines.FactsOf`: ephemeral, running or unread, held, a server down), read the same by its call and by every page that asks it. A judged VM carries the levels its gestures ask (`Asks`), which the window draws in place of the call's table by kind.
- **The answer.** `--yes` answers a yes or no and never a typed question. The name answers both, since it names exactly what the act touches, so `--confirm=infra` still stops a server.
- **A no is no error.** A declined act prints `Nothing changed.` with no `Error:`, and exits 1, ADR-021 rule 7's code for a declined act.
- **The window.** The delete dialog names what goes (`Delete linux-1 and its 40 GB disk?`), its red button stays red while it waits for the typed name, and a running VM offers `Stop instead`.
- **The proof.** `specs/architecture/consequence_test.go` pins the table apart from the code: every action graded, at worst, exactly as pinned, each consequence asking its one level, and every call whose verb destroys or stops declaring a confirmation. Its mutant moves `erase` to reversible and `stop` to destroys, and must be caught. `engine/feeds/confirm_test.go` holds the menu bar to its calls' levels.

## Consequences

- A script that passed `--yes` to a stop keeps working; one that passed `--confirm=<server>` to a stop or a restart too. A script that deletes a running ephemeral VM needs `--confirm=<name>` (Q2a).
- No flag and no verb changes. Which lifetime a create keeps by default is the breaking version's (M3), not this record's.
- A create whose image is missing builds it with no question, in a script too, and says so before the build starts, naming the image and that it takes minutes: the build stops nothing and loses nothing.
- Forgetting a device asks its name typed: it loses no data, but its add again needs its key and the owner at this Mac, so from afar, where a forget is meant to drop a stolen device, a wrong name is not given back (ADR-092 rule 3).
- Removing a password asks yes or no: its grants end, and the window asks it by the entry's label while the call checks its ID, so typing it waits on one name for both.
