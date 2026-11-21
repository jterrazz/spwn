# ADR-066: A commit owes that it builds and lints, a lot owes the gate

**Status:** Accepted
**Date:** 2026-11-21

Accepted by the owner on 2026-11-20.

## Context

A lot is a unit of work of one maker: one or more commits, each a unit of meaning. Until 2026-11-20 every commit was held to the whole gate on its own tree, alone, again at every restack: `make generate` with its drift, `go test`, the stories. On 2026-11-20 that loop ran 100 times to stack 79 commits their lots had already proved, 17.8 lane-hours and 685 min of wall clock. One commit's run, 10.5 min, cost as much as a cold gate of the whole tip. It found 7 defects: 4 were red at the tip, where one gate of the tip finds them; 3 were commits that were not green alone while their tip was green, two of them a `make lint` finding of 18 s, none a defect a tip gate would have shipped. Of 327 runs on 11-19 and 11-20, 100 were red on the drift or the stories alone, the shape of the known flakes.

## Decision

- **Every commit passes the cheap tier alone**: `gofmt`, `go vet` (linux, the world build tag, the specs module), `golangci-lint`, `make lint`, and the world build. About a minute, no story slot, several commits at once. It answers whether the commit builds and lints on its own tree, which is what `git bisect`, a revert and a cherry-pick need. The darwin-only files are vetted once, with cgo, by the Mac proof at the lot's tip, not per commit.
- **Every lot's tip, and the train's tip ([ADR-067](067-one-train-appended-by-makers-proved-at-its-tip.md)), passes the full gate**: `make generate` with drift, `go test`, the stories, the desktop proofs, and the Mac proofs when the lot moves what they prove.
- A behaviour change still lands with its test in the same commit; the gate at the tip is what runs it.

## Consequences

- A commit inside a lot may be red on a story while its lot's tip is green. A bisect is exact over lots, and over build and lint, but not over the stories inside a lot.
- The per-commit loop costs about a minute a commit instead of 10.5, and a stack of the size of 2026-11-20 saves about 500 lane-minutes.
- Rejected: the full gate at the tip alone, which loses "builds alone"; squashing a lot into fewer commits, which breaks the unit of meaning; carrying each step's verdict by the digest of its inputs, which rests on an input list not yet proved.
