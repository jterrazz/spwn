# Research Lab

> Same brain, new soul.

A patient, methodical agent named Curie. She keeps a real lab notebook - hypotheses, methods, observations, conclusions - and writes playbooks as she figures things out.

This example showcases **agent forking** - once Curie has learned enough, fork her into Darwin and watch him specialize differently. Same starting knowledge, divergent evolution.

## What's inside

| Component | Details |
|---|---|
| **World** | `research-lab` - 4 CPU, 4 GB RAM, 8 GB disk, 2h timeout |
| **Tools** | Unix, Git, Node.js 20, Python 3 |
| **Agent: curie** | Worker role. Careful, note-taking, hypothesis-driven. Documents everything in her journal. Writes playbooks from successful experiments. |

## Prerequisites

- spwn installed (`curl -fsSL https://spwn.sh/install.sh | bash`)
- Docker running
- An Anthropic API key (set via `claude setup-token` or `ANTHROPIC_API_KEY`)

## Install

```bash
spwn init @spwn/research-lab
```

## Spawn

```bash
# Give Curie a codebase to study
spwn up -c research-lab --agent curie -w ./my-project
```

## Explore

```bash
# Ask Curie to investigate something
spwn agent talk curie "Analyze the performance of the database queries in this project"

# Curie will:
#   1. State a hypothesis
#   2. Design an experiment
#   3. Run it
#   4. Record observations in her journal
#   5. Write conclusions to her knowledge

# Read her lab notebook
spwn agent journal curie

# Check her accumulated knowledge
spwn agent mind curie
```

## The forking experiment

Once Curie has built up knowledge from a few sessions:

```bash
# Consolidate what she's learned
spwn agent dream curie

# Fork her into a new agent
spwn agent fork curie darwin

# Now run both on different problems
spwn up -c research-lab --agent curie -w ./project-a
spwn up -c research-lab --agent darwin -w ./project-b

# Over time, they'll specialize differently
# - same starting knowledge, divergent playbooks
```

## What to try next

```bash
# Compare the two agents after they've diverged
spwn agent mind curie
spwn agent mind darwin

# Let them consolidate independently
spwn agent dream curie
spwn agent dream darwin
```

## Cleanup

```bash
spwn down <world-id>
rm ~/.spwn/worlds/research-lab.yaml
rm -rf ~/.spwn/agents/curie ~/.spwn/agents/darwin
```
