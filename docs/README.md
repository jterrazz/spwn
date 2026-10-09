# spwn — the manual

Everything this repository knows about itself, by what the reader came to do. What spwn is, why it exists, and its quickstart, is the README at the root of the repository; new here, [Getting started](start/getting-started.md) comes next.

## Start

| Page                                        | Answers                                                                                  |
| ------------------------------------------- | ---------------------------------------------------------------------------------------- |
| [Getting started](start/getting-started.md) | What a project holds, and the four ideas: one file per agent, the repo, worlds, runtimes |
| [Use cases](start/use-cases.md)             | Four sessions from an empty directory, end to end                                        |

## Guides

| Page                                             | Answers                                                                    |
| ------------------------------------------------ | -------------------------------------------------------------------------- |
| [Develop](guides/develop.md)                     | The toolchain, the loop, which file a change opens, and what a change owes |
| [Test](guides/test.md)                           | What proves a change, how to run each suite, and how to add a test         |
| [Release](guides/release.md)                     | The release runbook — signing keys, the tag that ships, the rollback       |
| [Automations](guides/automations.md)             | Waking an agent on a cron tick or a filesystem event                       |
| [Recipes](guides/recipes.md)                     | Worked examples, end to end                                                |
| [Run a manual pass](guides/run-a-manual-pass.md) | The three scenario catalogs a person walks against a live runtime          |

## Concepts

| Page                                       | Answers                                                                      |
| ------------------------------------------ | ---------------------------------------------------------------------------- |
| [Architecture](concepts/architecture.md)   | The monorepo layout, the seven layers imports flow down, DooD, the non-goals |
| [Domain model](concepts/domain-model.md)   | The vocabulary, the IDs, how an agent evolves                                |
| [Worlds](concepts/worlds.md)               | How a world runs, the Backend port, what crosses the boundary                |
| [Physics](concepts/physics.md)             | Constants, laws, elements, and the world context an agent reads at startup   |
| [The Mind](concepts/mind.md)               | Identity, skills, memory; Dream, Sleep, and versioning a Mind                |
| [Gate](concepts/gate.md)                   | The host-side broker: cookies, MCP routing, the browser sidecar              |
| [Observatory](concepts/observatory.md)     | The web UI and the API behind it                                             |
| [Update system](concepts/update-system.md) | How spwn ships and updates itself, from the git tag to the binary swap       |

## Reference

| Page                                                                | Answers                                                                     |
| ------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| [CLI](reference/cli.md)                                             | The `spwn <noun> <verb>` grammar and the command map                        |
| [Commands](reference/commands.md)                                   | Every command, projected from the Cobra tree by `make docs`                 |
| [Primitives](reference/primitives.md)                               | `spwn.yaml`, agents, tools, skills, hooks, commands, the dependency grammar |
| [Dependency catalog](reference/dependency-catalog.md)               | The built-in `spwn:*` entries and how to author your own                    |
| [Test architecture](reference/test-architecture.md)                 | Every suite, what it proves, the runners and the simulators                 |
| [Manual pass: CLI scenarios](reference/manual-cli-scenarios.md)     | Fifty command sequences the harness drives                                  |
| [Manual pass: agent behaviour](reference/manual-agent-behaviour.md) | Fifty scenarios that need a live, authenticated runtime                     |
| [Manual pass: edge cases](reference/manual-edge-cases.md)           | Seventy-two cases of concurrency, partial failure and filesystem edges      |

## Decisions

The records of what this repository decided, and why — the stack, the layering, the container model — are [decisions/](decisions/), one `NNN-subject.md` each.
