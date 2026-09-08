# ADR-015: Lint-Enforced Layered Imports

**Date:** 2026-04-16
**Status:** Accepted

> **Retroactive record.** Written 2026-09-04 during the knowledge harvest,
> from the enforcing commit (`6811986f`, "Enforce 5-layer architecture:
> packages/pack/, depguard, docs", 2026-04-16) and the session memory of
> the same date.

## Context

After the clean-slate reimplementation, the Go monorepo grew many domain
packages whose boundaries existed only in prose. Nothing stopped a
foundation package from importing a runtime package; layering violations
were invisible until they calcified. The refactor that created
`packages/pack/` (absorbing the pack schema, refs, and lockfile from
`image/` and `project/`) forced the question: what keeps the boundaries
honest afterwards?

## Decision

**Package imports flow downward only, and the rule is machine-enforced,
never aspirational.** Three mechanisms, layered themselves:

1. **Go `internal/`** — compiler-enforced visibility. Each module's
   implementation hides under `internal/`; only the root API is importable.
2. **depguard** (in `.golangci.yml`) — lint-time layer rules. Each layer
   carries deny rules for every package above it, so an upward import
   fails CI.
3. **The layer diagram in the corpus** — the human-readable map
   ([Architecture](../01-architecture.md)), rewritten as the roster churns.

At decision time the roster had five layers (foundation → domain →
project → build → runtime, with the CLI free above); it has since grown.
The invariant is the **direction and the enforcement**, not the layer
count: `.golangci.yml` and [Architecture](../01-architecture.md) own the
current graph.

## Consequences

### Positive

- **Boundaries survive refactors.** A violating import is a red build,
  not a code-review hope.
- **The map cannot lie silently.** When the lint config and the docs
  disagree, CI notices the config side.
- **Refactors get cheaper**—`packages/pack/` could absorb three scattered
  concerns because the deny rules pinned what may depend on what.

### Negative

- **Two sources to keep aligned**: the depguard rules and the prose
  diagram. The lint rules are the mechanical truth; the doc follows.
- **Layer churn has a cost**—every roster change edits deny lists across
  `.golangci.yml`.

## Alternatives Considered

| Alternative                     | Why Rejected                                                       |
| ------------------------------- | ------------------------------------------------------------------ |
| **Convention only (docs)**      | Was the status quo; violations accumulated invisibly               |
| **Single Go module, no layers** | Loses compiler-enforced `internal/` boundaries between domains     |
| **Import-graph test in Go**     | Hand-rolls what depguard already does inside the existing lint run |
