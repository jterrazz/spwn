# CLI

Spwn has one command line: the program `spwn`, whose every command follows one grammar. This page owns that grammar; [the glossary](../../machines/docs/concepts/glossary.md) owns every word in it, and [ADR-021](../decisions/021-one-program-spwn-one-command-for-everything.md) is the record that decided it.

| Section                  | Answers                                                          |
| ------------------------ | ---------------------------------------------------------------- |
| The grammar              | The nine rules every command holds                               |
| The help                 | The four groups, and which noun sits in which                    |
| Reaching another machine | `--on` and `--all-machines`                                      |
| Interim: two programs    | What runs today, and where each command goes                     |
| The command reference    | Where each program's every command is listed, and how it is made |

## The grammar

1. **Noun first.** `spwn <noun> [<verb>] [<name>...] [--flags]`. No verb stands at the top level, except the words that act on Spwn itself: `init`, `use`, `update`, `doctor`, `uninstall`, `help` and `--version`. Bare `spwn` is this machine's overview.
2. **A noun is a collection or a singleton.** A collection is plural (`machines`, `agents`, `apps`, `teams`); a singleton is not (`engine`, `disk`, `journal`, `machinefile`, `project`). A bare noun reads. `spwn <noun> <name>` is one entry's detail. `ls`, `list`, `show`, `inspect` and `status` exist nowhere.
3. **One verb per meaning, from one closed table, and no alias.** The table is [the glossary's verbs](../../machines/docs/concepts/glossary.md#the-verbs).
4. **An existing thing is a positional; a new thing is described by options.** A value that could be a secret is never an argv flag.
5. **The file is the input.** A Machinefile and a project's `spwn.yaml` each take `plan`, `apply` and `drift`. No command edits the declared grouping for you, so changing reality means changing the file and applying it.
6. **Four shared flags.**
    - `--json` on every read, answering `spwn.<noun>.<verb>/v1`.
    - `--yes` on an action that asks; `--confirm=<name>` on one that destroys named data.
    - `--on <machine>`.
    - `--all-machines` on a read that spans machines.
    - `--harness` names an agent's harness; `--backend` and `--runtime` do not exist.
7. **One failure shape and one exit table.** `Error:` and a pasteable `Hint:`, as the `spwn.error/v1` document under `--json`. Exit 0 is done, 1 failed or declined, 2 refused before any effect, 130 interrupted.
8. **A command stands alone.** What it needs is its refusal, and the hint is the command that provides it. No command starts a long-running process: the engine runs what runs.
9. **One vocabulary, one guard, read over the whole tree.** One conformance suite walks the one tree and fails on a verb outside the table, an alias, a read without `--json`, a destroying leaf without `--yes` or `--confirm`, a visible leaf without an Example, a noun in no help group, and a `--json` failure that is not `spwn.error/v1`.

Two invariants follow from rule 5 and rule 8. Commands speak the domain's words, never a backend's. And the declared file is the input while what runs is the output: no command changes a running thing in place.

## The help

The help has four groups: the three spaces of [ADR-019](../decisions/019-the-product-is-three-spaces-machine-agent-and-os.md), then Spwn itself.

| Group       | Holds                                                                                                                           |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------- |
| **Machine** | `machines`, `images`, `engine`, `approvals`, `apps`, `settings`, `disk`, `ports`, `accounts`, `background`, `awake`, `activity` |
| **Agent**   | `agents`, `teams`, `skills`, `automations`, `project`                                                                           |
| **OS**      | no command yet, so the help draws nothing for it                                                                                |
| **Spwn**    | `init`, `use`, `update`, `doctor`, `uninstall`, `help`, `--version`, and the `machinefile`, `journal` and `overview` reads      |

Where an agent runs, and how its project and its Mind follow it, is decided: an agent runs on a machine (`spwn agents start neo --on linux-1`), its project and Mind live on that machine, and the noun `world` is gone ([ADR-021](../decisions/021-one-program-spwn-one-command-for-everything.md), decisions 4 and 5). What an agent is given on that machine is [ADR-022](../decisions/022-an-agent-is-given-apps-its-machine-is-the-wall.md)'s.

## Reaching another machine

`--on <machine>` runs any command on another machine: this binary routes the call to the `spwn` on that machine, which runs it, and this binary renders the answer, so `--json` is identical wherever the call ran. `spwn use <machine>` makes every later command of one terminal run there. `spwn project` acts on the folder where it runs, so it refuses `--on`.

Another machine is reached through the `spwn` on it, never through an OpenSSH the user keeps ([Machines' ADR-090](../../machines/docs/decisions/090-a-machine-is-reached-through-its-engines-door-never-through-openssh.md)). Interactive `spwn agents talk <agent> --on <machine>` is launched by `spwn` itself, in the agent's Herdr terminal there ([Machines' ADR-089](../../machines/docs/decisions/089-an-agent-runs-in-a-herdr-terminal-on-its-own-machine.md)). How the binary is built, with Agents' commands as families of the engine, is [Machines' ADR-088](../../machines/docs/decisions/088-agents-live-in-the-one-binary-as-families-reached-by-calls.md).

## Interim: two programs

The grammar above is decided, and built only in part. Today the install puts two programs on the path, and each command exists once, in one of them; as a command moves, it is deleted from the other.

| Program   | Holds today                                                                                                                                                                                                                                      | Grammar                          |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------- |
| `station` | The Machine space: `machines`, `apps`, `settings`, `disk`, `ports`, `accounts`, `images`, `engine` and the rest of [Machines' commands](../../machines/docs/reference/commands.md), and `agents`, `skills`, `project`, `teams` and `automations` | The grammar, apart from the name |
| `spwn`    | Every other Agent command, as the map below lists them                                                                                                                                                                                           | The old grammar, below           |

`station help` draws the four groups: Machine and Spwn hold its commands, Agent holds `station agents`, an agent's Mind (its list and detail, `create`, `edit`, `delete`, `fork`, `export`, `import`, `dream`, `sleep`), `station skills`, its skill files (the list and a skill's text, `create`, `delete`), `station project`, the project of this folder (its file and agents, `create`, `plan`, `apply`, `drift`), `station teams`, the teams declared for you, and `station automations`, the automations the project declares and the runs written for them, these two read only, and OS holds no command yet, so the help does not draw it. The order decided is: the file half first (`agents`, `teams`, `skills`, `automations`, `project`), then the run half (`agents start`, `stop`, `talk`, `log`), then the old `spwn` is deleted and `station` takes the name `spwn`. Until the last step, `spwn` means Agents' program and the names `spwn-agents` and `spwn-machines`, with the shell aliases `agents` and `machines`, are the installer's old names: [ADR-021](../decisions/021-one-program-spwn-one-command-for-everything.md) voids them, and they never ship again once it is built.

### Where today's Agent commands go

Decided in ADR-021 and ADR-022. A row whose left column is `station` is built, under that name until `station` takes the name `spwn`; every other row is not built until its command is a family of the engine.

| Today                                             | The grammar's form                                                                                                                         |
| ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `station project create`, `--from <example>`      | `spwn project create`, `--from <example>`; `spwn init` (deleted) is gone                                                                   |
| `station project plan\|apply\|drift`              | `spwn project plan`, `spwn project apply` and `spwn project drift`                                                                         |
| `spwn build` (deleted)                            | gone: `spwn project apply` renders each agent into its run folder; the world image leaves `build`                                          |
| `spwn up`, `spwn down`, `spwn world *`            | `spwn agents start`, `spwn agents stop`, on a machine (`--on`); the noun `world` goes                                                      |
| `spwn agent neo`, `spwn agent talk neo`           | `spwn agents talk neo`                                                                                                                     |
| `station agents create\|fork\|dream\|sleep`       | `spwn agents create\|fork\|dream\|sleep`                                                                                                   |
| `station agents`, `station agents neo`            | `spwn agents`, `spwn agents neo`                                                                                                           |
| `station agents delete neo --confirm=neo`         | `spwn agents delete neo --confirm=neo`                                                                                                     |
| `station skills`, `station skills create\|delete` | `spwn skills`, `spwn skills create\|delete`; `skill edit` goes, a skill is its file                                                        |
| `spwn team *` (hidden)                            | `station teams` reads; a team is a file, `~/.spwn/teams/<team>.yaml`, and an agent joins it by `station agents edit <agent> --team <team>` |
| `spwn organization` (hidden)                      | gone                                                                                                                                       |
| `spwn automation ls\|status\|logs`                | `station automations` and `station automations log` read them, with no follow                                                              |
| `spwn automation daemon`                          | gone when the engine runs automations, which it does only while Spwn is open; until then it stays                                          |
| `spwn gate`, `spwn auth`, `spwn cookie-sync`      | `spwn accounts`, the one credential noun with one vault                                                                                    |
| `spwn web`                                        | gone: Spwn.app is the one GUI                                                                                                              |
| `spwn upgrade`                                    | `spwn update`                                                                                                                              |

`spwn project plan` does not check a `spwn:<name>` reference against the built-in catalog: the engine links no copy of it (`packages/dependency/builtin` carries the embedded catalog, and no package the engine links imports it), so such a reference passes the plan unverified, and the `unknown-builtin-tool` and `invalid-dependency` checks of the deleted `spwn check` are lost. `spwn install` still refuses an unknown name until it goes, and the rule that refuses every `spwn:` reference replaces it.

`station project apply` renders each agent into its run folder, `.spwn/agents/<agent>/`, and makes the whole declaration true, so what the deleted `spwn build` offered is lost on purpose: `--output` and `--force` (the folder is fixed; apply removes only what it wrote and asks before it overwrites a file edited by hand), `--agent` (no filter), `--runtime` (an agent declares its harness, and a sign-in never picks one), `--world` (each agent renders in the first grouping that holds it), `--dry-run` (`station project plan` lists what apply would write), and the image: `--tag`, `--base`, `--no-cache` and `SPWN_BASE_IMAGE` have no heir in the engine. `spwn up` builds its world image through the architect until `spwn up` is deleted, and an agent on a machine runs in its run folder, with no image.

`station project create` makes a new folder, `./<name>`, from the scaffold or, with `--from`, from one of the catalog's six examples, so what the deleted `spwn init` offered is lost on purpose: a project in the current folder, and over a `spwn.yaml` already there (`--force`; create refuses a folder that exists), `--name` (the folder is the name), `--backend` and its `spwn:` form (`--harness` names a harness bare, and an example's agents declare their own), the `spwn:<example>` form of an example (the name is bare), and the report of the worlds and agents an example added. The smoke pass, `make test-smoke`, no longer builds a real image of each example, since a spec reads only the fixtures of its own tree; `packages/catalog` still checks that every `spwn:` dependency an example declares is a tool of the built-in catalog.

`station automations` keeps what `spwn automation ls`, `status` and `logs` showed, and drops four things on purpose: `logs --follow` and its `-n` (no command follows a file, rule 8, so `--last` bounds the runs), the age of the last run (`3h ago`, which the page replaces by the instant), the daemon-lock rollup of `status` (the engine runs the automations later, and only while Spwn is open), and the 120-character cut of an error (the page shows its first line whole, the `--json` document all of it).

### Agents' commands today

```bash
# ── Project workflow ─────────────────────────────────────────────
station project create site   # a new folder ./site: spwn.yaml + agents/ skills/ ... + .spwn/ ignored
station project plan          # validate the tree
station project apply         # render each agent into .spwn/agents/<agent>/ (what plan lists)
spwn up [name]                # bring up every world in spwn.yaml (or one by name)
spwn agent neo                # spawn the world that contains neo and open a session
spwn down [name]              # stop every world (or one)

# ── Agents ───────────────────────────────────────────────────────
station agents create neo     # create a blank agent in ./agents/
station agents                # list project agents
station agents neo            # inspect composition, memory, history
station agents fork neo --as neo-v2  # clone memory + composition
station agents delete neo --confirm=neo  # delete an agent
spwn agent talk  neo "..."    # interactive session
station agents dream neo      # analyze experience, promote playbooks
station agents sleep neo      # consolidate memory, prune stale patterns

# ── Dependencies (compose) ───────────────────────────────────────
spwn install python                            # catalog dep, every agent
spwn install python --agent neo                # catalog dep, only neo
spwn install skill/paper-reading --agent neo   # local skill, only neo
spwn install tool/ffmpeg --agent neo           # local tool, only neo
spwn install hook/pre-spawn --agent neo        # local hook, only neo
spwn install command/refactor --agent neo      # local command, only neo
spwn uninstall python --agent neo              # detach from one agent
station skills create|delete <name>           # bare-markdown skill authoring

# ── Worlds ───────────────────────────────────────────────────────
spwn world list               # list running worlds
spwn world inspect <id>       # inspect a running world
spwn world enter   <id>       # interactive shell inside the world

# ── Automations ──────────────────────────────────────────────────
spwn automation daemon        # run the engine until interrupted

# ── System ───────────────────────────────────────────────────────
spwn gate start|stop|status   # host-side broker for cookie-bearing tools (see the Gate concept)
spwn auth login|logout|token  # provider credentials

# ── Registry (planned) ───────────────────────────────────────────
spwn install github:acme/fuzzer         # install from GitHub      [planned]
```

These follow the old grammar: noun then verb, `rm` for delete, `ls` for list, `up` and `down` at the top. They are what the table above retires.

## The command reference

Each program's every command is listed by a page generated from its own command tree, never hand-edited:

- `station`: [Machines' commands](../../machines/docs/reference/commands.md).
- `spwn`, today's Agents: [Commands](commands.md), regenerated with `make docs`.

## Related

- [The glossary](../../machines/docs/concepts/glossary.md): every word of the grammar, and the closed table of verbs.
- [Concepts](../concepts/domain-model.md): what the Agent nouns mean.
- [Primitives](primitives.md): the files an agent and a project are made of.
- [Automations](../guides/automations.md): the automation subsystem in depth.
