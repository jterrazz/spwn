# Recipes

Five real use cases. Each one is a working project you can paste into
your shell. The README has the concept pitch; this page is the how.

For the CLI surface see [`reference/spwn.md`](../reference/commands.md); for the list of
built-in dependencies see [Dependency catalog](../reference/dependency-catalog.md).

---

## 1. Compose a scientist from blocks

An autonomous lab partner: Python for code, QMD for local search over
your notebooks, and a skill that teaches the agent how to read papers.

```bash
station project create lab && cd lab
station agents create curie
spwn install python --agent curie
spwn install qmd --agent curie
spwn install skill/paper-reading --agent curie
spwn up
spwn agent talk curie "reproduce the results in notebooks/exp-042.qmd and flag anomalies"
```

What happened:

- `spwn install python --agent curie` adds `spwn:python` to
  `spwn/agents/curie/agent.yaml#dependencies:` **and** pins it in
  `spwn.lock`. Same for `spwn:qmd`.
- `spwn install skill/paper-reading --agent curie` scaffolds
  `spwn/skills/paper-reading.md` (empty; fill it in) and attaches the
  `skill/paper-reading` ref to the agent. At build time the skill is
  staged to `/world/skills/paper-reading/SKILL.md` inside the image;
  at spawn time Claude Code picks it up through its native
  `.claude/skills/` discovery path (symlinked from the agent's home).
- `station project create lab` made the folder `./lab` with its
  `spwn.yaml` and the starter agent `neo`; `station agents create curie`
  added `curie` beside it.
- `spwn up` materialises every world in `spwn.yaml`.
- `spwn agent talk` opens a session inside the running container with
  the workspace mounted.

**Iteration knobs.** Edit `SOUL.md` to change the voice. Add more
skills under `spwn/skills/`, each one a plain markdown file. Swap
the runtime in `agent.yaml#runtime.backend` when the Codex adapter
lands.

---

## 2. Ship an agent with your repo

Your teammates clone the repo, run `spwn up`, and get the same agent
byte-for-byte: no setup docs, no onboarding slides.

```bash
station project create acme-api && cd acme-api
spwn install node --agent neo
spwn install git --agent neo

git add spwn.yaml spwn.lock spwn/
git commit -m "add neo, our repo maintainer"
git push
```

What happened:

- `spwn.yaml` + `spwn.lock` + `spwn/` are now part of your repo like
  any other code. PRs that change `agent.yaml` dependencies show up
  as normal diffs.
- `spwn.lock` pins every dep's version so two clones of the repo
  produce the same image.
- The next teammate runs `spwn up` and is talking to the same neo you
  were: same tools, same `SOUL.md`, same playbooks.

**What's persistent, what's not.** `spwn/agents/<name>/` is the mind:
commit it. `~/.spwn/` holds per-machine credentials and daemon
state; never commit that. Sessions and journal entries written
inside a running container sync back out on graceful `spwn down`;
destructive shutdowns lose them.

---

## 3. Fork a mind, throw it away if it breaks

`git checkout -b` for agents. Try a risky refactor in a branch; keep
the agent if it worked, delete it if it didn't.

```bash
station agents fork neo --as neo-migration       # clone composition + memory
spwn up --agent neo-migration
spwn agent talk neo-migration "migrate the whole repo from Jest to Vitest"

# worked?  keep neo-migration.
# didn't?  neo is untouched, no regrets.
station agents delete neo-migration --confirm=neo-migration
```

What happened:

- `fork` copies `spwn/agents/neo/` to `spwn/agents/neo-migration/`
  (manifest, `SOUL.md`, playbooks, journal) and adds a single-agent
  world for the clone.
- `spwn up --agent neo-migration` boots the fork into its own
  container: neo and neo-migration run in parallel without touching
  each other's state.
- `station agents delete neo-migration --confirm=neo-migration` deletes the directory and the world
  entry. If anything from the experiment is worth keeping, copy the
  playbooks back into neo before removing.

**Why this matters.** Destructive work against a real codebase used
to mean "trust one agent not to break things." Now it means "branch,
try, merge or bin."

---

## 4. Read untrusted code inside a container

Clone a repo you don't trust and ask an agent what it does, inside a
Docker container rather than in your shell.

```bash
station project create sus && cd sus
git clone https://github.com/someone/sus-repo code
spwn up                    # a Docker container on the bridge network, 256 processes at most
spwn agent talk neo "run every test and benchmark, tell me what the code actually does"
```

What happened:

- The world is a container on Docker's bridge network, with outbound
  access: the code inside can reach the internet.
- Its one resource limit is 256 processes. CPU, memory, disk and time
  are not capped, so a runaway process takes what the Docker host
  gives it.
- The repo is mounted under `/workspaces/` read-write; the rest of
  your disk stays outside the container, except the other mounts the
  world declares.

**Limits today.** A world is not a sealed room: no `cpu:`, `mem:` or
network knob exists, and the code can change the mounted repo. Don't
use it for code you believe is hostile; a VM is the stronger boundary,
and running VMs is [Machines](../../machines/README.md)' job.

---

## 5. Two agents collaborating in one world

Multi-agent worlds put more than one mind in the same container,
talking to each other through a shared inbox filesystem.

```yaml
# spwn.yaml
version: 1
name: sample-colony

worlds:
    matrix:
        agents: [morpheus, neo]
        workspaces: [.]
```

```bash
station agents create morpheus
station agents create neo
# edit spwn.yaml to list both under one world (see above)
spwn up
spwn agent talk morpheus "delegate the API audit to neo"
```

What happened:

- Both agents' home directories are copied into the same container at
  `/agents/morpheus/` and `/agents/neo/`. They share one process
  namespace and one workspace mount.
- Inter-agent messages are plain markdown files written to
  `/world/inbox/<recipient>/<timestamp>-from-<sender>.md`. Each agent
  polls its own inbox: no queue, no broker, just files.
- Roles (chief, worker) are declared in each agent's `agent.yaml`
  under `role:`. The Architect tier can be wired to orchestrate
  larger colonies from the outside.

**When to use.** Splitting work across specialized minds
(reviewer + implementer, architect + worker). For a single lone
agent, one world with one agent is simpler; don't force
multi-agent until the division of labour is real.
