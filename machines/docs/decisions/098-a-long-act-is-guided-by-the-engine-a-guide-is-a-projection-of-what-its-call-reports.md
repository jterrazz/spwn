# ADR-098: A long act is guided by the engine; a Guide is a projection of what its call reports

**Status:** Proposed
**Date:** 2026-11-26

Builds on [ADR-070](070-the-engine-keeps-one-read-model.md) (store, projection, view) and [ADR-062](062-a-failed-create-stays-on-the-record-until-its-start-or-its-delete.md) (a failed create stays on the record).

## Context

A create takes minutes and crosses steps (an image's build, the clone, the boot, the address, SSH) that the window must show as one Guide: the step it runs, what failed, what was kept and the one repair. Each client wording that itself would give the act as many owners as there are clients, and an act a client drove would end with the client's window. The create already reports its steps as lines through its call's Progress, and the VM's record already keeps a failed birth.

## Decision

- **The call stays the only driver of its act.** A call whose `Meta` names a `Guide` family reports each step it reaches as a typed `guide.Phase` through its Progress (`guide.Report`); it holds no state machine of its own.
- **The engine keeps facts, not views.** Its server tees a guided call's Phases to a small board (`process.Acts`, persisted to `acts.json`): each act's rail, step, line and end. The call's end ends the act: `done`, `failed` with the call's error, or `interrupted` when the call is cancelled or its owner process no longer runs. An ended act leaves the board 10 minutes after it ended, a fact the board writes.
- **A Guide is a projection.** One pure function per act family (`machines.CreateGuide`) turns an act and the record of its subject into a `guide.Guide`, served as the `engine.state` view `engine.guides`, and as the call of the same name. The words are the engine's.
- **A repair is a call, never a command.** A Guide's gesture names a call and its input, or a page; it carries no command line.
- **A refused repair is the act's state.** A call whose `Meta` names the act it `Repairs` and that the engine refuses ends that act failed again on the refusal's code and words, from the moment of the refusal; the projection draws it as it draws any failure, so the window computes no words of it.
- **The first frame that names an act carries its Guide.** A guided call's progress frames name their act and step; the one that first names the act also carries the act's Guide, projected from the act as the board put it, and no later frame repeats it, so a client that started the act draws its Guide at once, before its list holds the act.
- **A step reaches the store at once.** Each write of the board puts its slice into the store, so a step reaches a watcher without waiting for the readers' tick.
- **The words are one list.** The pictures and statuses a Guide names are closed enums generated into `packages/api`, and the rule `guides` holds them to the window's component, and the guided families of `calls.json` to their projections.

## Consequences

- A Guide survives the window: closed and reopened, the window reads the same act from the engine; the act lives as long as its call, never as long as a client.
- A create's progress is live lines, never a measure: its steps have no honest total of work.
- Only a create asked of an engine's socket is guided: a create a command runs itself (`station machines create`), and a server's create, report no Phase that anyone keeps.
- The call is `engine.guides`, not the plan's `guides.list`: a call of the engine's package is named by its family (rule `calls-gated`).
- A family gains a Guide by naming it on its call and adding its projection to `feeds.Guided`; the rule fails one without the other.
- A repair names the machine it runs at (`On`, as a finding's fix does), and a made create's Guide carries one action, the page of its machine (`Nav`), so the window wires gestures and draws nothing else of them; its heading is the engine's words, never the family's id. A create asked of another Mac is not guided, since only the engine whose socket the window asked runs the act's board.
- Not yet decided here: re-running a create that failed before its first run (the plan's `guides.again`), which today offers no repair.
