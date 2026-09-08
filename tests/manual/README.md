# Manual QA

Three scenario catalogs a human — or a human-driven Claude session — walks end to end. They are not part of `make test`: they run against a real Anthropic-authenticated runtime, real Docker and real disk, and they exercise the paths the automated suite skips on purpose because they cost money, are non-deterministic, or are about what an agent *perceives* rather than what a process exits with.

The strategy these serve, and the ten rules every test holds to, are [`docs/03-testing.md`](../../docs/03-testing.md); the automated layer pyramid is [`../ARCHITECTURE.md`](../ARCHITECTURE.md).

## The suites

| Suite                                    | Scenarios | Driver               | Probes                                                                                                                            |
| ---------------------------------------- | --------- | -------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| [`cli-scenarios.md`](cli-scenarios.md)   | 50        | `harness.sh` (bash)  | Realistic command sequences a user runs in a day — init → build → up → talk → down. Uses `mock-claude`; mostly automatable.        |
| [`agent-behavior.md`](agent-behavior.md) | 50        | Human + live Claude  | Whether the host-side setup — CLAUDE.md, playbooks, skills, hooks, tools, knowledge, roster, messaging — actually reaches the agent. |
| [`edge-cases.md`](edge-cases.md)         | 72        | Mixed (bash + human) | Concurrency, partial failure, state-machine holes, filesystem edges, tool-install failure modes.                                    |

Only `agent-behavior.md` needs a live session: it asks what the agent *sees*, and `mock-claude` sees nothing.

## Running a pass

```bash
make build         # .artifacts/go/spwn
make test-image    # spwn-test:latest, the mock-runtime image

SPWN=$PWD/.artifacts/go/spwn bash tests/manual/harness.sh        # all 50
SPWN=$PWD/.artifacts/go/spwn bash tests/manual/harness.sh 1 15   # a subset
```

The harness exits non-zero with a failure count. All user-level state lives under `$SPWN_HOME` (default `/tmp/qa-50/spwn_home`) and each scenario gets its own scratch directory, so a real `~/.spwn` is never touched.

## What a pass owes

A pass is not a document to file — it is a list of bugs. Each one it surfaces lands as a fix **and** the automated guard that would have caught it, in the same commit; a scenario that cannot be automated says why, inline, so the debt is visible where it is incurred. Nothing here keeps a dated scoreboard: what a pass found is the commits it produced, and git already tells that story.

If a pass turns up a category none of the three suites covers, it earns a fourth catalog rather than an overloaded existing one.
