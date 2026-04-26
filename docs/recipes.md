# Recipes

Five real use cases. Each one is a working project you can paste into
your shell. The README has the concept pitch; this page is the how.

For the CLI surface see [`cli/spwn.md`](cli/spwn.md); for the list of
built-in dependencies see [`dependency-catalog.md`](dependency-catalog.md).

---

## 1. Compose a scientist from blocks

An autonomous lab partner: Python for code, QMD for local search over
your notebooks, and a skill that teaches the agent how to read papers.

```bash
spwn init
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
- `spwn up` materialises every world in `spwn.yaml` — by default,
  `spwn init` gave you one single-agent world named `curie`.
- `spwn agent talk` opens a session inside the running container with
  the workspace mounted.

**Iteration knobs.** Edit `SOUL.md` to change the voice. Add more
skills under `spwn/skills/` — each one is a plain markdown file. Swap
the runtime in `agent.yaml#runtime.backend` when the Codex adapter
lands.

---

## 2. Ship an agent with your repo

Your teammates clone the repo, run `spwn up`, and get the same agent
byte-for-byte — no setup docs, no onboarding slides.

```bash
cd acme-api
spwn init
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
  were — same tools, same `SOUL.md`, same playbooks.

**What's persistent, what's not.** `spwn/agents/<name>/` is the mind:
commit it. `~/.spwn/` holds per-machine credentials and daemon
state — never commit that. Sessions and journal entries written
inside a running container sync back out on graceful `spwn down`;
destructive shutdowns lose them.

---

## 3. Fork a mind, throw it away if it breaks

`git checkout -b` for agents. Try a risky refactor in a branch; keep
the agent if it worked, delete it if it didn't.

```bash
spwn agent fork neo neo-migration       # clone composition + memory
spwn up --agent neo-migration
spwn agent talk neo-migration "migrate the whole repo from Jest to Vitest"

# worked?  keep neo-migration.
# didn't?  neo is untouched, no regrets.
spwn agent rm neo-migration
```

What happened:

- `fork` copies `spwn/agents/neo/` → `spwn/agents/neo-migration/`
  (manifest, `SOUL.md`, playbooks, journal) and adds a single-agent
  world for the clone.
- `spwn up --agent neo-migration` boots the fork into its own
  container — neo and neo-migration run in parallel without touching
  each other's state.
- `spwn agent rm neo-migration` deletes the directory and the world
  entry. If anything from the experiment is worth keeping, copy the
  playbooks back into neo before removing.

**Why this matters.** Destructive work against a real codebase used
to mean "trust one agent not to break things." Now it means "branch,
try, merge or bin."

---

## 4. Unleash untrusted code in a sealed room

Clone a repo you don't trust, ask an agent what it does — inside a
container with no network and hard resource caps.

```bash
git clone https://github.com/someone/sus-repo /tmp/sus && cd /tmp/sus
spwn init
spwn up                    # no network, CPU/mem/disk/time caps
spwn agent talk neo "run every test and benchmark, tell me what the code actually does"
```

What happened:

- The spawned container inherits Docker host defaults for CPU,
  memory, and disk — enough for real work, capped so a runaway
  process can't melt the host.
- No network interface means nothing in the container can phone home,
  fetch payloads, or exfiltrate data.
- The repo is mounted under `/workspaces/` read-write; everything else
  on your machine is invisible.

**Limits today.** Per-world hard limits (explicit `cpu:` / `mem:`
knobs) are planned but not wired; you get the Docker defaults. Don't
use this for code you actively believe is hostile — it's a brake,
not a vault.

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
spwn agent create morpheus
spwn agent create neo
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
  polls its own inbox — no queue, no broker, just files.
- Roles (chief, worker) are declared in each agent's `agent.yaml`
  under `role:`. The Architect tier can be wired to orchestrate
  larger colonies from the outside.

**When to use.** Splitting work across specialized minds
(reviewer + implementer, architect + worker). For a single lone
agent, one world with one agent is simpler — don't force
multi-agent until the division of labour is real.
