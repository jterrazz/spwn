# Startup

> 3 worlds · 3 agents · one company.

A complete miniature company running across three parallel worlds.
This example exists to show what spwn feels like when you go beyond
a single world — each environment has its own physics, its own agent,
and its own lifecycle.

## What's inside

- **World** `prod` · **Agent** `ceo` — the decision-maker. Reads the
  research world's conclusions, reads devops' health reports, and
  picks what ships.
- **World** `staging` · **Agent** `devops` — keeps the pipe flowing.
  Lives in a tighter-constrained world with shorter timeouts and
  narrower tools.
- **World** `research` · **Agent** `analyst` — explores unproven
  ideas in isolation. No production access, no deployment rights.

## Try it

Install, then spawn each world in its own terminal:

```sh
spwn up -c prod --agent ceo
spwn up -c staging --agent devops
spwn up -c research --agent analyst

spwn ls
# w-prod-12345      ● ceo      · running
# w-staging-67890   ● devops   · running
# w-research-00021  ◌ analyst  · idle
```

Then let the CEO read the others' journals:

```sh
spwn agent talk ceo "read the latest from analyst and devops, then decide what ships this week"
```

## Remove

```sh
rm ~/.spwn/worlds/{prod,staging,research}.yaml
rm -rf ~/.spwn/agents/{ceo,devops,analyst}
```
