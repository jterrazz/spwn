# ADR-091: An adopted app is a state, never work

**Status:** Proposed
**Date:** 2026-11-25

Amends [ADR-034](034-apply-carries-a-machinefile-the-ledger-bounds-removal.md).

Decided by the orchestrator under the owner's delegation of 2026-11-25.

## Context

Applying a Machinefile on a Mac left four casks (Claude, Discord, Notion, Jump Desktop), present but not installed by brew, listed under `To adopt` by the next `plan` and `not managed` by `drift`, run after run. Adoption records nothing: [ADR-034](034-apply-carries-a-machinefile-the-ledger-bounds-removal.md)'s ledger holds what apply installed, and bounds removal by it. Reading them as work was the defect, since apply does nothing for them.

## Decision

- **A declared app present outside its method is adopted, and adopted is a state.** The plan lists it under `Adopted`, as information, and counts nothing to do for it; a plan holding only adopted apps is calm.
- **Drift counts it as matching.** Its row keeps the fact (`not managed`); it is no difference and no verb of what a bare apply does.
- **Nothing is recorded and nothing handed over.** The ledger keeps bounding removal to what apply installed, and no `brew install --cask --adopt` runs: an app that updates itself stays the owner's.
- **An app the file does not list keeps its reading,** `not managed` in `station apps`.

## Consequences

- The plan after an apply that installed everything is calm, and drift reads `Matches`.
- An app never brought under any manager stays visible on the plan, and no longer hides in the count of work.
- `Applying` loses its `Adopt` count and the sentence its `adopt` verb.
