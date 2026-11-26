# ADR-099: `project apply` writes only what its plan lists, and removes only what it wrote

**Status:** Proposed
**Date:** 2026-11-26

Builds on [ADR-088](088-agents-live-in-the-one-binary-as-families-reached-by-calls.md): the project family is a family of the engine, reached by its calls. Root [ADR-021](../../../docs/decisions/021-one-program-spwn-one-command-for-everything.md) retires `spwn build`; this record says what replaces its render.

## Context

`spwn build --tree-only` wrote an agent's rendered tree into an output folder, and with `--force` emptied that folder first, whatever it held. Its default built a Docker image from the tree. On a machine, an agent runs in a run folder of the project's gitignored state, `.spwn/agents/<agent>/`, with links into its Mind beside the rendered files, so a writer that empties the folder would delete what it never wrote. `project plan` already rendered each agent in memory to judge it; a second render path for the write could disagree with it.

## Decision

1. **One Render hook.** The contract `Projects.Render(root, agent)` returns an agent's files, each a path, its bytes, its mode and whether it is a link; it writes nothing. Both `project plan` and `project apply` compute their changes from it through one pure function, `project.ChangesOf`, so what a plan lists is what an apply writes.
2. **The harness is declared.** An agent renders for its own `runtime.backend`, else the project's, else the catalog's first harness; never one picked by a sign-in.
3. **A record of what was written.** Apply keeps the digest of each file it wrote in `.spwn/applied.json`. A file it wrote and the declaration no longer renders is removed; a file it never wrote is never read nor touched.
4. **A confirmation follows its consequence.** An apply that only adds asks nothing. One that overwrites or removes names each file, a file whose digest is not the record's as changed by hand, and asks once, yes or no; off a terminal and with `--json` it refuses without `--yes`. A plan with an error writes nothing. A second apply writes nothing.
5. **No image.** Apply builds no image and starts no agent.

## Consequences

- R1a's `agents start` renders through the same hook into the same folder; the links to the Mind are more files of the hook's answer, so neither its signature nor the writer changes.
- `spwn build`'s `--output`, `--force`, `--agent` and `--runtime` have no successor: the folder, the scope and the harness are the declaration's.
- Drift does not yet tell a file changed since apply; plan and apply say it, as `changed by hand`.
