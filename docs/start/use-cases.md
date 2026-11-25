# Use cases

Four sessions that show what Spwn is for, each a few commands from an empty directory. The ideas behind them are [Getting started](getting-started.md).

## Compose a scientist from blocks

```bash
spwn init
spwn install python --agent curie
spwn install qmd --agent curie
spwn install skill/paper-reading --agent curie
spwn up
spwn agent talk curie "reproduce the results in notebooks/exp-042.qmd and flag anomalies"
```

> Stack `spwn:python` + `spwn:qmd` + the right skills and you have an autonomous lab partner. Edit `SOUL.md` tomorrow - same mind, new voice. **Docker, but for minds.**

## Ship an agent with your repo

```bash
cd acme-api
spwn init
spwn install node --agent neo
spwn install git --agent neo

git add spwn.yaml spwn/
git commit -m "add neo, our repo maintainer"
git push

# every teammate who clones the repo gets the same mind, byte-for-byte
```

> Agents orchestration as code, shared like code. PR-review a behavior change. Bisect an agent's memory like bisecting a bug. **The Dockerfile metaphor, all the way.**

## Fork a mind, throw it away if it breaks

```bash
spwn agent fork neo neo-migration       # clone composition + memory
spwn up --agent neo-migration
spwn agent talk neo-migration "migrate the whole repo from Jest to Vitest"

# worked? promote.  didn't? neo is untouched, no regrets.
spwn agent rm neo-migration
```

> The only AI assistant that lets you `git checkout -b` your agent. Run a destructive refactor in a branch; keep or discard based on the diff. **Natural selection for behavior.**

## Read untrusted code inside a container

```bash
git clone https://github.com/someone/sus-repo /tmp/sus && cd /tmp/sus
spwn init
spwn up                    # a Docker container on the bridge network, 256 processes at most
spwn agent talk neo "run every test and benchmark, tell me what the code actually does"
```

> The code runs in a Docker container, not in your shell, and on your disk it reaches only the mounts the world declares. It is not a sealed room: the world has outbound network through Docker's bridge, the repo is mounted read-write, and its one resource limit is 256 processes, with CPU, memory, disk and time uncapped. A VM is the stronger boundary, and running VMs is [Machines](../../machines/README.md)' job.
