# Research Lab

> Local skills, local tools, and scientific rigor.

A meticulous research agent named Curie that investigates questions using
empirical methods. She designs experiments, analyzes data in Jupyter
notebooks, and writes reproducible Quarto reports.

This example demonstrates the **new dependency model**: project-wide deps,
agent-specific local skills, and local tools.

## What this example demonstrates

### Local skills (`spwn/skills/`)

Three research methodology skills that the agent can invoke:

- **literature-review** - systematic search and synthesis of existing work
- **experiment-design** - controlled experiment design with pre-registered analysis plans
- **data-analysis** - data cleaning, statistical testing, and visualization

Skills are Markdown files with detailed procedural instructions. They live
in `spwn/skills/` and are referenced by name in `agent.yaml`.

### Local tools (`spwn/tools/`)

- **jupyter** - a local tool that installs Jupyter via pip. Defined in
  `spwn/tools/jupyter/pack.yaml` with install commands and a verify step.

### Dependency model

- `spwn.yaml` declares **project-wide deps** (`@spwn/python`, `@spwn/qmd`)
  that every agent inherits automatically.
- `agent.yaml` declares only **agent-specific additions** (skills and tools).
  It does not repeat the project deps.
- `spwn.lock` records all resolved dependencies in a flat, line-oriented
  text format.

## What's inside

```
research-lab/
  spwn.yaml                       # Project config: name, deps, worlds
  spwn.lock                       # Resolved dependency versions
  agents/
    curie/
      agent.yaml                  # Agent config: skills, tools
      identity/
        profile.md                # Persona: meticulous scientist
      AGENTS.md                   # Provider-neutral system prompt
  spwn/
    skills/
      literature-review.md        # Skill: search and synthesize papers
      experiment-design.md        # Skill: design controlled experiments
      data-analysis.md            # Skill: statistical analysis + viz
    tools/
      jupyter/
        pack.yaml                 # Tool: install and verify jupyter
```

## Install

```bash
spwn init @spwn/research-lab
```

## Spawn

```bash
spwn up -c research-lab --agent curie -w ./my-dataset
```

## Example interactions

```bash
# Ask Curie to investigate a question
spwn agent talk curie "Is there a significant difference in response time between v2 and v3 of the API?"

# Curie will:
#   1. Review existing documentation and benchmarks (literature-review)
#   2. Design a controlled benchmark experiment (experiment-design)
#   3. Execute the experiment and analyze results in Jupyter (data-analysis)
#   4. Write a conclusion with statistical evidence

# Read her lab notebook
spwn agent journal curie

# Check accumulated playbooks
spwn agent mind curie
```

## Cleanup

```bash
spwn down <world-id>
```
