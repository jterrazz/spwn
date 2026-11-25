# ADR-022: An agent is given apps, and its machine is the wall

**Status:** Accepted
**Date:** 2026-11-24

An agent leaves the status `Proposed`; only the owner writes `Accepted`.

Accepted by the owner on 2026-11-24, in the answers quoted below.

## Context

Today a tool is shared by a whole world, not given per agent: a project has one image, built from the union of its agents' dependencies, and agents sharing a world must align versions. Only the local blocks (`skill/`, `hook/`, `command/`) are chosen per agent. The invariant is "Composition, not restriction. Security comes from what the world image lacks" (`docs/reference/primitives.md`).

The world leaves: an agent runs on a machine ([ADR-021](021-one-program-spwn-one-command-for-everything.md)). On a machine there is no wall between the agents that share it: the VM user has passwordless sudo, apps installed by apt live in `/usr/bin`, an agent with a shell and a network can fetch any program, and every agent runs under the same uid. Machines' catalog already holds the apps (`machines/catalog/packages/`), refuses for a missing app and offers to install it, and delivers a credential per program: `gh` into its own store, `npm` into `.npmrc`, `claude` and `wrangler` as environment.

Asked whether an agent's tool is a catalog app, the owner answered: « (remarque : en fait ça va remplacer ce concept, mais faudra trouver un moyen d'injecter ou non un outil de la machine dans l'agent.) » (2026-11-24). Asked then whether "given, with the machine as the wall" is what that means, or whether he wants a real wall between agents on a shared machine: « masque, la machine est le mur » (2026-11-24).

Weighed and left: the machine alone as the boundary (it does not answer "injecter ou non"); PATH alone (no wall, and presented alone it would claim more than it does). One unix user per agent, and a Linux mount namespace, are real walls kept for later: the first means user management on macOS and the end of `sudo apt` for agents, the second walls nothing on macOS.

## Decision

- **An agent lists the catalog apps it is given**, under a key `apps:` in `spwn/agents/<agent>/agent.yaml`, and a project may list a pool shared by all its agents in `spwn.yaml`, unioned in as dependencies are today. The word is "apps", the one `spwn apps` already uses.
- **The default is none.** An agent gets the base system, its harness and the harness's sign-in, and the apps it lists. Nothing else is passed to it.
- **The version is the machine's.** An agent cannot pin one: the machine's lock decides ([Machines' ADR-041](../../machines/docs/decisions/041-the-lock-is-what-apply-installs.md)). The app entries of `spwn.lock` go.
- **`spwn:<name>` and `tool/<name>` leave the dependency grammar.** `skill/`, `hook/` and `command/` stay the agent's own local files. The root catalog's tool entries are deleted, each mapped to its twin in Machines' catalog.
- **The target's `spwn` builds the harness's environment at the agent's start**, passed in Herdr's `terminal.run` ([Machines' ADR-089](../../machines/docs/decisions/089-an-agent-runs-in-a-herdr-terminal-on-its-own-machine.md)):
    1. **Presence.** A given app that is missing is a refusal that names it; on a terminal the kit offers to install it, with the pasteable hint `spwn apps install <app> --on <machine>`.
    2. **PATH.** A bin folder of the agent's own holds links to each given app's commands, then the system folders. Homebrew's, mise's shims, npm's globals and `~/.local/bin` are left out. The docs say "not on its path", never "denied".
    3. **Authority.** A given program's sign-in reaches the process. A program that is not given gets none: one read from the environment is never passed in, and one with its own store has its config variable pointed at an empty folder of the agent's. Those variables become catalog data, a field of the package's delivery.
    4. **Skills and context.** Given skill apps are linked into the agent's own harness folder, and its rendered context names only its given apps.
    5. **Honesty.** `spwn agents <agent>` lists the given, the missing, and the reachable but not given. `spwn project plan` reports a program signed in on the machine that an agent there is not given.
- **This is a mask, and the machine is the wall.** An agent that must not reach something runs on a machine that does not have it. No new kind of grant: an agent that needs authority of its own runs on a machine of its own, and a grant to that machine is its grant.
- **The commands**: `spwn agents edit <agent> [--app <app>]... [--no-app <app>]...` and `spwn project edit [--app <app>]... [--no-app <app>]...`; an app the catalog does not know is refused with exit 2 and `Hint: spwn apps --all`. `project install`, `project remove` and today's `spwn install` and `uninstall` are deleted, so `install` means one thing: onto a machine. `agents start` takes no `--app`: the file is the input.

## The twins

Each tool entry of the root catalog is deleted, mapped to its twin in Machines' catalog (`machines/catalog/packages/<category>/<name>.yaml`). A twin is installed by the machine's lock, never pinned by an agent.

| Root entry   | Twin                                               | Left behind                                                                                                    |
| ------------ | -------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| `architect`  | none                                               | The fleet daemon and its skills are not an app; the lot that deletes the world carries them                    |
| `build`      | `build-essential` (command-line)                   |                                                                                                                |
| `cli`        | none                                               | The machine's own `spwn` is the program; its skill is carried by the lot that deletes the world                |
| `docker-cli` | `docker` (containers-vms)                          |                                                                                                                |
| `gh`         | `gh` (git)                                         | `gh-mcp` and `@modelcontextprotocol/server-github`: the CLI is the whole app                                   |
| `git`        | `git` (git)                                        |                                                                                                                |
| `mcp2cli`    | `mcp2cli` (ai-tooling), new                        | `spwn-policy-check` and its per-agent method policy: they belong to the world image and leave with it          |
| `mempalace`  | `mempalace` (ai-tooling), new                      | Its skill: an agent's own local `skill/` file                                                                  |
| `node`       | `node` (runtimes)                                  | Node 20 pinned by an agent: the machine's lock decides the version                                             |
| `notion`     | `notion-mcp` (ai-tooling), new                     | The host-bind-mounted token cache: the account is the machine's. `notion` stays the desktop app                |
| `python`     | `python` (runtimes)                                | The `python` link and `pip3`: the app provides `python3`                                                       |
| `qmd`        | `qmd` (ai-tooling), and `qmd-skill` (agent-skills) |                                                                                                                |
| `unix`       | none                                               | The base system carries bash, coreutils, grep, sed, awk and curl; `jq` (command-line) is the one an agent asks |
| `x`          | none yet                                           | Its gate element waits for H2's gate                                                                           |

## Consequences

- The invariant "Security comes from what the world image lacks" is rewritten: security comes from what the machine lacks; what an agent is given is a mask over it.
- An app installed by apt stays reachable to an agent that is not given it, and a program fetched by the agent is its own. The honesty rows say so; they never claim a wall.
- `docs/reference/primitives.md`, `docs/reference/dependency-catalog.md` and the root catalog's tool entries are rewritten or deleted by the lot that builds this; until then they describe the world that ships.
- The config variables read by probe P4 (2026-11-24): `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `GH_CONFIG_DIR`, `NPM_CONFIG_USERCONFIG`; `wrangler` reads only `XDG_CONFIG_HOME`, so its mask moves every XDG program's folder with it.
- Open when this was written: whether a per-agent `CLAUDE_CONFIG_DIR` keeps the Mind's sessions (probe PH2).
- A real wall between agents on one shared machine (a unix user each, or a mount namespace on Linux) stays a later decision of its own.
