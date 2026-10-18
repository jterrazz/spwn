# Run a manual pass

Three scenario catalogs a human (or a human-driven Claude session) walks end to end. They are not part of `make test`: they run against a real Anthropic-authenticated runtime, real Docker and real disk, and they exercise the paths the automated suite skips on purpose because they cost money, are non-deterministic, or are about what an agent _perceives_ rather than what a process exits with.

What the automated suites prove, and where, is [Test architecture](../reference/test-architecture.md).

## The suites

| Suite                                                     | Scenarios | Driver                      | Probes                                                                                                                             |
| --------------------------------------------------------- | --------- | --------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| [CLI scenarios](../reference/manual-cli-scenarios.md)     | 50        | `scripts/manual-harness.sh` | Realistic command sequences a user runs in a day: init, build, up, talk, down. Uses `mock-claude`; mostly automatable.             |
| [Agent behaviour](../reference/manual-agent-behaviour.md) | 50        | Human + live Claude         | Whether the host-side setup (CLAUDE.md, playbooks, skills, hooks, tools, knowledge, roster, messaging) actually reaches the agent. |
| [Edge cases](../reference/manual-edge-cases.md)           | 72        | Mixed (bash + human)        | Concurrency, partial failure, state-machine holes, filesystem edges, tool-install failure modes.                                   |

Only the agent-behaviour catalog needs a live session: it asks what the agent _sees_, and `mock-claude` sees nothing.

## Running a pass

```bash
make build         # .artifacts/go/spwn
make test-image    # spwn-test:latest, the mock-runtime image

bash scripts/manual-harness.sh        # all 50
bash scripts/manual-harness.sh 1 15   # a subset
```

The harness drives `.artifacts/go/spwn` unless `SPWN` names another binary, and exits non-zero with a failure count. All user-level state lives under `$SPWN_HOME` (default `/tmp/qa-50/spwn_home`) and each scenario gets its own scratch directory, so a real `~/.spwn` is never touched.

## What a pass owes

A pass is not a document to file: it is a list of bugs. Each one it surfaces lands as a fix **and** the automated guard that would have caught it, in the same commit; a scenario that cannot be automated says why, inline, so the debt is visible where it is incurred. Nothing here keeps a dated scoreboard: what a pass found is the commits it produced, and git already tells that story.

If a pass turns up a category none of the three suites covers, it earns a fourth catalog rather than an overloaded existing one.
