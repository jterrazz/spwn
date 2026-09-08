# spwn docs

The written corpus for spwn — every piece of knowledge lives here exactly once, and the root brief and the README vitrine route in without restating a word.

| Chapter                                            | Holds                                                                              |
| -------------------------------------------------- | ---------------------------------------------------------------------------------- |
| [01 Architecture](01-architecture.md)              | The monorepo layout, the seven layers imports flow down, DooD, the non-goals.       |
| [02 Developing](02-developing.md)                  | The toolchain, the loop, which file a change opens, and what a change owes.         |
| [03 Testing](03-testing.md)                        | The layer pyramid, the two spec forms, the manual passes, what a test may assume.   |
| [04 Operating](04-operating.md)                    | The release runbook — signing keys, the tag that ships, the rollback.               |
| [05 Concepts](05-concepts.md)                      | The domain model, the vocabulary, the IDs, how an agent evolves.                    |
| [06 CLI](06-cli.md)                                | The `spwn <noun> <verb>` grammar and the command map.                               |
| [07 Primitives](07-primitives.md)                  | `spwn.yaml`, agents, tools, skills, hooks, commands, and the dependency grammar.    |
| [08 Gate](08-gate.md)                              | The host-side broker: cookies, MCP routing, the browser sidecar.                    |
| [09 Worlds](09-worlds.md)                          | How a world runs, the Backend port, what crosses the boundary.                      |
| [10 Physics](10-physics.md)                        | Constants, laws, elements, and the world context an agent reads at startup.         |
| [11 The Mind](11-mind.md)                          | Identity, skills, memory; Dream, Sleep, and versioning a Mind.                      |
| [12 Observatory](12-observatory.md)                | The web UI and the API behind it.                                                   |
| [13 Automations](13-automations.md)                | Waking an agent on a cron tick or a filesystem event.                               |
| [14 Recipes](14-recipes.md)                        | Worked examples, end to end.                                                        |
| [15 Dependency catalog](15-dependency-catalog.md)  | The built-in `spwn:*` entries and how to author your own.                           |
| [16 Update system](16-update-system.md)            | How spwn ships and updates itself, from the git tag to the binary swap.             |
| [decisions/](decisions/)                           | The decision records — why the stack, the layering, the container model.            |
| [reference/](reference/)                           | The generated per-command pages, projected from Cobra by `make docs`.               |
