# ADR-002: Defer Instincts Layer to v2

**Date:** 2025-02-10
**Status:** Accepted

## Context

During Mind framework design, we identified four types of agent memory:

| Type          | Academic Term       | Purpose                      |
| ------------- | ------------------- | ---------------------------- |
| Knowledge     | Semantic memory     | Facts and understanding      |
| Playbooks     | Procedural memory   | How-to instructions          |
| Journal       | Episodic memory     | What happened                |
| **Instincts** | **Implicit memory** | **Auto-discovered patterns** |

Instincts are behavioral patterns that emerge from repeated experience: things the agent "just knows" without explicit instruction. For example, an agent that has debugged 50 Node.js projects might develop an instinct to always check `node_modules` freshness first.

The question: should we add a fourth agent profile directory (`instincts/`) in v1?

## Decision

**Defer the instincts layer to v2.** In v1, auto-discovered patterns are stored in `playbooks/` with an `auto-discovered: true` frontmatter tag.

```yaml
# playbooks/check-node-modules-freshness.md
---
auto-discovered: true
confidence: 0.82
source: journal analysis (2025-02-15)
---
When debugging a Node.js build failure, first check if node_modules
is stale by comparing package-lock.json mtime to node_modules mtime.
```

When the volume and sophistication of auto-discovered patterns justifies a dedicated directory, we promote `instincts/` in v2.

## Rationale

Three directories is the right starting point:

1. **We don't know the shape of instincts yet.** Are they playbook fragments? Weighted heuristics? Behavioral policies? We need real-world data before committing to a structure.
2. **Premature abstraction is costly.** Adding a directory is easy. Removing one (and migrating content) is hard.
3. **The tag approach is flexible.** `auto-discovered: true` lets us query and filter without structural commitment.
4. **Trinity is elegant.** Knowledge/Playbooks/Journal maps cleanly to semantic/procedural/episodic memory. Adding a fourth weakens the metaphor unless it earns its place.

## Consequences

### Positive

- Simpler v1: three directories, three concepts, easy to explain.
- Auto-discovered patterns still have a home (playbooks with a tag).
- We gather real data before designing the instincts structure.

### Negative

- Playbooks directory may accumulate noise as auto-discovered entries mix with hand-written ones.
- The `auto-discovered` tag is informal: no schema enforcement in v1.

### Mitigations

- Naming convention: auto-discovered playbooks use a `_auto-` prefix in the filename.
- When playbooks/ has more than 20 auto-discovered entries, revisit this ADR.

## Alternatives Considered

| Alternative                | Why Rejected                                                              |
| -------------------------- | ------------------------------------------------------------------------- |
| **Add `instincts/` now**   | Premature: we don't know the right structure yet                          |
| **Store in `journal/`**    | Journal is chronological; instincts are derived patterns, not events      |
| **Store in `knowledge/`**  | Knowledge is factual; instincts are behavioral, a different category      |
| **Flat file in Mind root** | Unstructured, doesn't scale, breaks the directory-per-memory-type pattern |
