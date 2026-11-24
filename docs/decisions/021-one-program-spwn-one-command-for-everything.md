# ADR-021: One program, `spwn`, one command for everything

**Status:** Accepted
**Date:** 2026-11-24

An agent leaves the status `Proposed`; only the owner writes `Accepted`.

Accepted by the owner on 2026-11-24, in the answers quoted below.

Amends [ADR-017](017-one-repository-station-moves-in-under-machines.md): its installed names and its "Later" bullets.

## Context

Spwn ships two command lines. The engine's (`station`, under `machines/`) is noun first, answers `--json` with versioned documents, and has one failure shape and one exit table. Agents' (`spwn`, `apps/cli`) has top-level verbs, five spellings of "bring a world up", aliases and daemons of its own. ADR-017 installed both under long names (`spwn-agents`, `spwn-machines`) with shell aliases, and left open whether one umbrella command exists. Machines' ADR-076 had already chosen one name, `spwn`, for the engine on the path.

A review of both trees (2026-11-24) proposed one grammar, nine rules, four help groups, and two executables behind one front door. The owner answered it, then a second round on running one program, then a third on Herdr and on reaching other machines.

## Decision

1. **One program, one linked binary.** `spwn` is the only command a user types, on every machine. Agents' commands become families of the engine, reached by calls and rendered by the engine's kit. No `spwn-agents`, no `spwn-machines`, no `agents` or `machines` alias ever ships. How the binary is built is Machines' [ADR-088](../../machines/docs/decisions/088-agents-live-in-the-one-binary-as-families-reached-by-calls.md).
2. **Nine rules hold every command.**
    1. Noun first: `spwn <noun> [<verb>] [<name>...] [--flags]`. No verb at the top level, except the words that act on Spwn itself: `init`, `use`, `update`, `doctor`, `uninstall`, `help`, `--version`. Bare `spwn` is this machine's overview.
    2. A noun is a collection (plural: `machines`, `agents`, `apps`, `teams`) or a singleton (`engine`, `disk`, `journal`, `machinefile`, `project`). A bare noun reads; `spwn <noun> <name>` is one entry's detail. `ls`, `list`, `show`, `inspect` and `status` exist nowhere.
    3. One verb per meaning, from one closed table, and no alias: `create`/`delete`, `start`/`stop`/`restart`, `install`/`update`/`remove`, `plan`/`apply`/`drift`, `add`/`forget`, `edit`, `login`/`logout`, `log`, and a noun's own acts (`talk`, `dream`, `sleep`, `fork`, `share`, `free`, `erase`, `build`, `ssh`), each defined once in the glossary.
    4. An existing thing is a positional; a new thing is described by options. A value that could be a secret is never an argv flag.
    5. The file is the input. A Machinefile and a project's `spwn.yaml` each take `plan`, `apply` and `drift`. No command edits the declared grouping for you.
    6. Four shared flags: `--json` on every read (`spwn.<noun>.<verb>/v1`); `--yes` on an action that asks, `--confirm=<name>` on one that destroys named data; `--on <machine>`; `--all-machines` on a read that spans them. `--backend` and `--runtime` become `--harness`.
    7. One failure shape and one exit table: `Error:` and a pasteable `Hint:`, `spwn.error/v1` under `--json`; exit 0 done, 1 failed or declined, 2 refused before any effect, 130 interrupted.
    8. A command stands alone. What it needs is its refusal, and the hint is the command that provides it. No command starts a long-running process: the engine runs what runs.
    9. One vocabulary, one guard, read over the whole tree.

    One conformance suite walks the one tree and fails on a verb outside the table, an alias, a read without `--json`, a destroying leaf without `--yes` or `--confirm`, a visible leaf without an Example, a noun in no help group, and a `--json` failure that is not `spwn.error/v1`.

3. **The help has four groups**, the three spaces of [ADR-019](019-the-product-is-three-spaces-machine-agent-and-os.md) and Spwn itself: **Machine**, **Agent**, **OS**, **Spwn**. OS shows one muted line, "being made", and no command until it runs.
4. **The place an agent runs in leaves the Agent space.** An agent runs on a machine (`spwn agents start neo --on linux-1`). The noun `world` and its commands go; the grouping is the team's, in the file.
5. **An agent's project and Mind live on the machine it runs on.** `--on` routes its calls there, the far `spwn` runs them, and this binary renders the answer. `spwn project` acts on the folder where it runs, so it refuses `--on`.
6. **Interactive `spwn agents talk <agent> --on <machine>` is launched by `spwn` itself**, through Herdr: it attaches this terminal to the agent's Herdr terminal on that machine. No hint to type another command, and no `ssh`. The terminal is Machines' [ADR-089](../../machines/docs/decisions/089-an-agent-runs-in-a-herdr-terminal-on-its-own-machine.md).
7. **Another machine is reached through the `spwn` on it**, never through OpenSSH the user has to keep: Machines' [ADR-090](../../machines/docs/decisions/090-a-machine-is-reached-through-its-engines-door-never-through-openssh.md).
8. **What an agent may use is given, app by app**, and its machine is the wall: [ADR-022](022-an-agent-is-given-apps-its-machine-is-the-wall.md).
9. **The six example projects become templates in the catalog** (`macrohard`, `matrix`, `paperclip-factory`, `research-lab`, `severance`, `startup`).
10. **The rest of the review is taken as recommended**: `accounts` is the one credential noun, with one vault (`auth`, `gate` and `cookie-sync` go); the engine runs everything that runs, automations included; the architect leaves the Agent space, and `talk` comes back as the OS space's first command; `spwn web` goes, Spwn.app is the one GUI.

### The owner's words

From the audit (2026-11-24):

- On the menu bar's absence (T2): « Absent (remarque : absent mais du coup bro je veux une app unique on avait dit nn ? ah peut être pas en fait. Mais on devrait revoir le projet nn ? Spwn command unique pour tout tout tout) »
- On reaching other machines (S1): « Corriger (remarque : spwn devrait se connecter aux autres ordis par le binaire qui expose le service spwn nn ? et rien de ssh à supporter etc etc.) »

On the command review (2026-11-24):

| Question                                                    | Answer                                                                                                                                   |
| ----------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| Q1. Does the place an agent runs in leave the Agent space?  | « oui »                                                                                                                                  |
| Q2. One command over two executables, or one linked binary? | « un seul programme »                                                                                                                    |
| Q3. No verb at the top level?                               | « oui »                                                                                                                                  |
| Q4. Is `accounts` the one credential noun, with one vault?  | « oui »                                                                                                                                  |
| Q5. Does the engine run everything that runs?               | « oui »                                                                                                                                  |
| Q6. Is an agent's tool a catalog app?                       | « (remarque : en fait ça va remplacer ce concept, mais faudra trouver un moyen d'injecter ou non un outil de la machine dans l'agent.) » |
| Q7. Is the architect the OS's pilot?                        | « oui »                                                                                                                                  |
| Q8. Does `spwn web` go?                                     | « oui »                                                                                                                                  |
| Lots                                                        | « M1, M2, A1, A2 »                                                                                                                       |

On running one program (2026-11-24):

| Question                                                                                 | Answer                                                                                                                                                     |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Q1. Is "given, with the machine as the wall" what "injecter ou non" means?               | « masque, la machine est le mur »                                                                                                                          |
| Q2. Where do an agent's project and Mind live when it runs `--on` another machine?       | « sur la machine cible (remarque : mais remplaçons certaines choses par Herdr-core , je pense qu'on a pas à tout gérer si Herdr peut le faire pour nous) » |
| Q3. Interactive `agents talk --on`: refuse with a pasteable hint, or have `spwn` run it? | « refuser avec le hint ssh »                                                                                                                               |
| Q4. Where do the six example projects live once `apps/cli` is gone?                      | « dans le catalogue »                                                                                                                                      |

On Herdr and the door (2026-11-24), the first answer replacing Q3 above, since ssh leaves:

| Question                                                              | Answer            |
| --------------------------------------------------------------------- | ----------------- |
| H1. Interactive `talk --on`: a hint, or does `spwn` launch it itself? | « spwn le lance » |

## Consequences

- ADR-017's installed names are void: one `spwn`, no long names, no aliases. Its "Later" bullets are settled: no rename to `spwn-machines` or `spwn-agents`, the umbrella command is the program itself, and responsibilities move by ADR-088.
- `apps/cli`, `specs/cli/` and `apps/api` are deleted once their commands live in the engine, lot by lot. Each command exists once at every step.
- The README's command blocks, `docs/reference/cli.md` (the owner of the grammar) and the one glossary are rewritten to this record by their own change; until then they describe the two command lines that ship.
- Whether a project may carry catalog packages of its own, as a third source, is not decided here.
- Open when this was written, each settled by its probe before the lot that rests on it starts:
    - H1, a machine carries what a world carries: holds on the mounts' uid mapping (probe P1b, 2026-11-24); the time to create a machine (about 8 s against Docker's 0.65 s) is a cost, not a refutation.
    - H2, the engine supervises the gate's work: unread.
    - H3, `--on` reaches an Agent command: holds by construction once the command is a call; proved by its spec.
    - H4, `check` folds into `plan` and `drift`: holds (probe P2); drift becomes a row that refuses nothing.
    - H5, the harnesses' sign-ins are accounts programs: holds for `claude`; `codex` with ChatGPT and Notion's MCP sign-in need a fourth delivery shape (probe P4).
    - H6, the binary stays small with Agents' modules linked: holds, +1.5 %, no Docker client (probe P3).
    - H7, a harness outlives the call that started it: carried by Herdr ([ADR-089](../../machines/docs/decisions/089-an-agent-runs-in-a-herdr-terminal-on-its-own-machine.md)), proved by its probes.
