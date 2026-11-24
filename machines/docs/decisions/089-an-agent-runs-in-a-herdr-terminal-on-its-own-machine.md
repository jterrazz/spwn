# ADR-089: An agent runs in a Herdr terminal on its own machine

**Status:** Accepted
**Date:** 2026-11-24

Accepted by the owner on 2026-11-24, on the owner's word.

## Context

An agent's harness must outlive the call that started it, on a Mac or a VM. Nothing does that today: Agents' `agents start` and `stop` are not implemented, and an interactive `talk` is a `docker exec -it`. Building it here would mean a systemd user unit and a pty wrapper on Linux, a launchd job on macOS, linger, and an attach stream with scrollback and resize. A probe on a fresh VM (P1b, 2026-11-24) found the traps: linger is off on a fresh image, tmux moves its pane out of the unit's cgroup, and survival without the engine's own guest session was left unproven.

Herdr already does it. herdr-core (`hc`, `herdrdev/herdr-core`, Apache-2.0, not the owner's code) keeps terminals alive across their viewers' disconnects and across its own restart and update, enables linger on Linux, attaches and detaches with scrollback, sends text and keys, reads the screen, and relays a far machine's terminals into one list for herdr-app and the phone. Its API is JSON-RPC 2.0 on a 0700 unix socket, versioned by protocol generation (today 1), and it only grows. It re-runs nothing after a reboot. herdr-core, herdr-app and herdr are already catalog apps.

[ADR-024](024-agents-run-in-herdr.md) once ran agents in Herdr from the engine, and [ADR-052](052-station-keeps-no-agents.md) took that out: "Herdr stays a package, never station's wiring". The owner answered, on where an agent runs (2026-11-24): « sur la machine cible (remarque : mais remplaçons certaines choses par Herdr-core , je pense qu'on a pas à tout gérer si Herdr peut le faire pour nous) ».

## Decision

- **Each machine's own herdr-core holds the agents that run on it.** That machine's `spwn` drives it, locally, through Herdr's API. Across machines it is `spwn`'s calls; Herdr's own machine link only puts a far terminal on this screen.
- **The `Session` contract**, in `apps/engine/agents/contract/`, is the boundary. Its real adapter, `agents/adapter/herdr`, speaks JSON-RPC on the machine's herdr-core socket, the endpoint `hc` itself resolves; `adapter/fake` stands for it in the world.

    | Session | Herdr's API                                                                                                                     |
    | ------- | ------------------------------------------------------------------------------------------------------------------------------- |
    | `Start` | `terminal.run` with `argv`, `cwd`, `env`, `name` (the agent), `labels` (`spwn.agent`, `spwn.project`), `no_parent`, `no_notify` |
    | `Stop`  | `terminal.kill`, then `terminal.rm`                                                                                             |
    | `Send`  | `terminal.send`                                                                                                                 |
    | `Read`  | `terminal.read`                                                                                                                 |
    | `List`  | `terminal.list`, filtered on the `spwn.agent` label                                                                             |

    Only `agents/adapter/herdr` speaks Herdr's protocol, held by a layers rule.

- **Text typed to an agent is never on a command line**, as ADR-024 had it: it goes through `terminal.send`.
- **`hello` is checked for protocol generation 1.** A missing or too-old herdr-core is `call.NeedMissing` with the package `herdr-core`, so the kit offers to install it.
- **`agents.start`, `stop`, `log` and `talk <message>` run at the target**: the far `spwn` talks to its own Herdr. `talk <message>` to an agent with no live terminal stays the harness's one-shot run.
- **Interactive `agents talk <agent>` attaches this terminal**: here by `hc attach <terminal>`, and with `--on <machine>` by `spwn` launching the attach to `<machine>/<terminal>` itself. The owner's answer: « spwn le lance ».
- **herdr-core is a required catalog app where an agent runs, never bundled.** It has its own updater, signing team and cadence, as tart is an outside dependency. The owner's answer: « Oui ». The images that run agents list `herdr-core` and turn linger on themselves, without relying on Herdr's own attempt.
- **`spwn` keeps what is the agent's**: its definition, Mind, teams and skills; where it runs; the environment built at its start ([the repository's ADR-022](../../../docs/decisions/022-an-agent-is-given-apps-its-machine-is-the-wall.md)), passed in `terminal.run`'s `env`; the harness command and its session id, so the agent carries on after a reboot; the record of agent to machine and terminal; grants, accounts, installing herdr-core, and every page it renders.
- **The dependency runs one way, `spwn` to Herdr.** Herdr never imports `spwn` and never calls its API.

### The owner's words

On Herdr and the door (2026-11-24):

| Question                                                                                | Answer            |
| --------------------------------------------------------------------------------------- | ----------------- |
| H1. Interactive `talk --on`: a hint, or does `spwn` launch it itself?                   | « spwn le lance » |
| H2. Where an agent runs, herdr-core is a required catalog app, never bundled in `spwn`? | « Oui »           |

## Consequences

- ADR-052's bullet "Herdr stays a package, never station's wiring" is reversed for the agents' terminals. Its two other halves hold: Herdr is installed as a package, and `spwn` never reaches a machine through Herdr.
- ADR-024 stays superseded; what still holds of it (no tmux, text never in argv, labels) is restated here.
- No systemd unit, pty wrapper or launchd job of `spwn`'s own supervises a harness.
- The window first shows "Open in Herdr" on an agent's row. An embedded viewer, a client of Herdr's `terminal.attach` channel, is a later decision; `spwn` never builds a pty stream protocol of its own.
- Status (working, blocked, idle, done) comes from Herdr only when the harness reports it. Whether `claude` does is unread.
- Probes owed before the lot that builds it: a herdr-core started on a fresh VM outlives the guest connection, the engine's ssh session and a logout (PH1); `terminal.run`'s `env` reaches the harness, and its session resumes after a reboot from `spwn`'s record (PH2).
