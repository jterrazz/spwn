# The Mind

An agent is three layers — identity, skills, memory — and the whole point of drawing the line between them is how they behave over time. This chapter is that design: what each layer holds, what Sleep and Fork do to it, and how an agent's memory is versioned. The on-disk shape of the files is [Primitives](07-primitives.md); the vocabulary is [Concepts](05-concepts.md).

## The three layers

- **Identity** — the immutable core: purpose, values, bonds, role, voice. It does not fork, does not sleep, does not prune.
- **Skills** — invocable capabilities: what the agent can do.
- **Memory** — the evolving layer: knowledge, playbooks, journal. It grows through experience, restructures during Sleep, and branches when the agent is forked.

The profile is mounted into every world the agent enters; without one, every world starts from zero. Identity is what keeps the accumulation coherent — an agent that evolves long enough without one drifts out of its purpose.

Security constraints never live in any of these layers. They belong to the world ([Physics](10-physics.md)).

### What each layer defines

| Layer     | What it defines                                          | The rule that is not obvious                                                     |
| --------- | -------------------------------------------------------- | -------------------------------------------------------------------------------- |
| Purpose   | Why the agent exists — mission plus the scope it owns    | A north star, not a task list: it is what a learning agent is anchored to        |
| Values    | What the agent stands for — core principles              | Character traits it upholds, never behavioural constraints                       |
| Bonds     | Who the agent trusts — operators and peers, with a level | Bonds survive forks, so a clone is never orphaned from its chain of accountability |
| Role      | Who the agent is — role, style, operating preferences    | Identity, not chains: the agent may deviate on judgment                          |
| Skills    | What the agent can do — self-contained capabilities      | Each carries a trigger, required context, steps, and a rollback                  |
| Knowledge | What the agent knows — facts, stack, conventions         | World-scoped rather than agent-scoped; mutable and queryable                     |
| Playbooks | How the agent acts — a "when" clause plus numbered steps | Hand-written or auto-discovered from journal patterns ([ADR-002](decisions/002-defer-instincts-layer.md)) |
| Journal   | What the agent experienced — one file per session        | Appended by the system: world ID, image, outcome, exit code, duration            |

### Skills, playbooks, and knowledge

The three memory-adjacent layers look alike and are not:

|              | Skills                     | Playbooks                        | Knowledge                  |
| ------------ | -------------------------- | -------------------------------- | -------------------------- |
| Purpose      | Define a capability        | Record a procedure               | Store facts                |
| Triggered by | Operator request or event  | A situation that matches         | Query or reference         |
| Structure    | Trigger, steps, rollback   | When, then steps                 | Free-form facts            |
| Mutability   | Versioned, hand-authored   | Hand-authored or auto-discovered | Continuously updated       |
| In one line  | "I can deploy"             | "When I deploy, I do X, Y, Z"    | "The deploy target is AWS" |

## Soul and Mind

The Soul is the identity layer: purpose, values, bonds. The Mind is everything that changes: skills, knowledge, playbooks, journal. The split is behavioural.

|              | Soul                             | Mind                                     |
| ------------ | -------------------------------- | ---------------------------------------- |
| Changes      | Never, or rarely and with intent | Constantly: grows, prunes, restructures  |
| During Sleep | Untouched                        | Consolidated, pruned, reorganised        |
| During Fork  | Shared across all forks          | Each fork gets its own copy              |
| During Merge | No conflict possible             | Needs conflict resolution                |

Because Claude Code is the runtime spawned inside the world ([ADR-004](decisions/004-claude-code-as-runtime.md)), these files are read natively as markdown: `SOUL.md` and `AGENTS.md` shape identity the way `.claude/agents/` does, `playbooks/` the way `.claude/skills/` does. There is no custom loader.

## Lifecycle and mount

The Architect creates a world, mounts the agent's profile into it, and generates the world's physics and faculties; the container-side gate spawns the agent CLI. The agent reads identity, skills, and knowledge to know who it is, then physics and faculties to know what is possible, and works — following playbooks, invoking skills, appending to the journal — until the world is destroyed.

The profile outlives the world. The mount is read-write: the agent writes knowledge as it learns, appends to the journal, and creates a playbook when it finds a repeatable procedure ([Worlds](09-worlds.md) for the mount mechanics).

## Dream — learning from one task

Dream is the incremental cycle, one task at a time: the agent executes a task and appends to its journal, reviews those entries, distils patterns into candidate playbooks or knowledge updates, and promotes the high-confidence ones with an `auto-discovered: true` tag. The next task starts from the promoted playbook. It is Reflexion (Shinn et al., 2023) applied to a persistent filesystem.

The framework supplies the primitives — snapshot, fork, merge — and the reflection logic runs as an agent-level feature on top.

## Sleep — restructuring the whole Mind

An agent that only accumulates degrades: the journal grows unbounded, new learnings contradict old ones, outdated playbooks go unreviewed, the self-model drifts. Sleep is the phase where the agent steps back from execution and restructures the entire Mind. Four operations, plus one optional:

1. **Consolidate** — compress journal entries into knowledge: raw detail fades, lessons persist.
2. **Prune** — retire low-confidence playbooks, remove stale knowledge.
3. **Reorganise** — merge related playbooks, resolve contradictions.
4. **Update the self-model** — revise the agent's own understanding of its strengths and weaknesses.
5. **Dream** (optional) — cross-reference disparate experiences for non-obvious patterns.

Sleep has depth: a nap compresses recent entries after a few tasks, light sleep prunes and consolidates after a day's work, deep sleep runs the full restructuring including the dream — weekly, or after a milestone. The Soul is the fixed point through all of it: Sleep can reorganise knowledge and playbooks completely and the agent stays recognisably itself.

### Open question: the Sleep execution model

Undecided — should Sleep run inside a world (the agent restructures its own Mind through the runtime: consistent and creative, but slow and token-hungry), outside it (a framework process over the profile directory: fast and deterministic, but unable to resolve semantic contradictions), or as a hybrid (the framework does journal compression and dedup, the agent does contradictions, self-model, and dreaming)? Leaning toward the hybrid: mechanical consolidation does not need an LLM; semantic restructuring does.

## Versioning a Mind

Memory is versioned the way code is: snapshot, fork, merge, rollback. This is what lets an agent try a risky approach without corrupting what it knows, run two strategies in parallel, or hand an experienced Mind to a new agent.

| Operation | What it does                                              | Effect on the Soul            |
| --------- | --------------------------------------------------------- | ----------------------------- |
| Snapshot  | Captures a Mind at a point in time; immutable afterwards  | Not included, always current  |
| Fork      | Copies a Mind so it can diverge; the original is untouched | Inherited, shared not copied  |
| Merge     | Applies one branch's changes into another                 | No conflict possible          |
| Rollback  | Restores a Mind to a snapshot, discarding what came after | Untouched                     |

Two forks of one agent share the same Soul, so however far they diverge in knowledge and skill they remain the same agent — they diverged in how they operate, not in why they exist.

Versioning is per agent, not per world: each agent with a Mind branches independently, which is what makes specialisation practical (fork a general-purpose worker into domain specialists and let each evolve) and what makes a failed approach cheap (roll one agent's Mind back to a known-good state without touching its peers in the same world).

### Merging

| Conflict  | Resolution                                                                |
| --------- | ------------------------------------------------------------------------- |
| Knowledge | Keep the most recent version, or have the agent reconcile the two         |
| Playbook  | Keep both as alternatives and let the agent choose on context             |
| Journal   | No conflict possible: journals are append-only, so a merge is concatenation |

Undecided — which of those becomes the default when two branches touched the same file: last-write-wins (simple, loses one side), agent-mediated merge (intelligent, non-deterministic, costs a model call), keep-both (no loss, duplicates accumulate), or human resolution (best quality, does not scale). Leaning toward agent-mediated for knowledge, keep-both for playbooks, concatenation for the journal.

### What is shipped

Dream, Sleep, and Fork ship today (`spwn agent dream|sleep|fork`), built on filesystem snapshots. Merge and richer branching are ahead; content-addressable storage is a later option if the snapshots get expensive.

Designed, not shipped: agents that fork, experiment, and merge without an operator in the loop.

## Future: the instincts layer

A fourth memory type — instincts, the implicit patterns that emerge from repeated experience — is planned for v2. Until then auto-discovered patterns live in `playbooks/` behind the `auto-discovered: true` tag ([ADR-002](decisions/002-defer-instincts-layer.md)).

## Related

- [Concepts](05-concepts.md) — Soul, Memory, Dream, Sleep, Fork in one line each.
- [Primitives](07-primitives.md) — the agent directory and `agent.yaml`.
- [Worlds](09-worlds.md) — how the profile is mounted and what persists.
