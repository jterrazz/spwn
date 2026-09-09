# ADR-011: Universe Manifest (`org.yaml`)

**Status:** Accepted
**Date:** 2026-03-29

## Context

As spwn scales from a single developer to teams and universes, several needs emerge:

1. **Shared defaults.** Every world and agent in a universe should share the same provider, backend, and physics defaults without repeating them in every manifest.
2. **Governance.** Universes need to enforce limits (max worlds, cost caps, allowed providers) and audit agent activity.
3. **Shared skills.** Teams need a common set of skills (coding standards, deployment procedures) available to all agents.
4. **Multi-channel Architect.** Architect needs to know which messaging channels to connect to and how to sync configuration.
5. **Config versioning.** Configuration changes---both by humans and by Architect---should be versioned and auditable.

Without a top-level manifest, these concerns are scattered across individual world and agent configs, duplicated, and hard to govern.

## Decision

**Introduce `org.yaml` as the top-level Universe manifest at `~/.spwn/org.yaml`.** It sits above worlds and agents in the configuration hierarchy.

### Manifest Hierarchy (Cascading Overrides)

```
org.yaml          → Universe defaults (source of truth)
  world.yaml      → Per-world overrides
    profile.yaml     → Per-agent overrides
```

Each level inherits all settings from the level above and can override any of them. Resolution order: `profile.yaml` > `world.yaml` > `org.yaml` > built-in defaults.

### What `org.yaml` Contains

| Section            | Purpose                                                                |
| ------------------ | ---------------------------------------------------------------------- |
| `defaults.runtime` | Default runtime backend, provider, model, and auth for all agents      |
| `defaults.backend` | Default Backend port adapter for all worlds                            |
| `defaults.memory`  | Default Memory port adapter                                            |
| `defaults.store`   | Default Store port adapter                                             |
| `defaults.physics` | Default physics (constants, laws, elements) for all worlds             |
| `skills`           | Shared skills available to all agents (marketplace, git, local)        |
| `governance`       | Limits and policies (max-worlds, cost-limit, allowed-providers, audit) |
| `claw.channels`    | Channel port adapters for Architect communication                      |
| `claw.sync`        | Git sync configuration for `~/.spwn/`                                  |

### Why Config Sync to Git

Architect manages itself using spwn (dogfooding). The `claw.sync` section configures auto-sync of `~/.spwn/` to a git repository:

- **User changes flow in.** A user edits `org.yaml` on GitHub. Architect pulls the change. Adapters reconfigure.
- **Architect changes flow out.** Architect promotes a citizen, updates a profile.yaml. The change is pushed. Visible in git history.
- **Rollback is trivial.** A bad configuration change is a `git revert` away.
- **Audit trail.** Every configuration change---by human or by Architect---is a git commit with a timestamp and author.

### Why Cascading Overrides

The cascading model is chosen over flat configuration because:

1. **DRY.** Set provider=anthropic once in `org.yaml`. 50 agents inherit it. Override one agent to use OpenAI.
2. **Governance.** The org manifest is the authority. It can restrict what lower levels can override (e.g., `governance.allowed-providers` limits which providers agents can use).
3. **Familiarity.** CSS cascading, Terraform variable precedence, Kubernetes admission controllers all use the same pattern.

## Consequences

### Positive

- **Single source of truth.** One file governs the entire universe.
- **Declarative governance.** Limits, policies, and audit are configuration, not code.
- **Git-native.** Config sync makes all changes versioned and auditable.
- **Composable.** org.yaml composes with the ports & adapters architecture---it declares which adapters to use at each level.
- **Natural hierarchy.** Universe > World > Agent mirrors real organizational structure.

### Negative

- **New concept to learn.** Users must understand cascading overrides.
- **Merge conflicts.** Multiple people editing org.yaml simultaneously can conflict.
- **Complexity.** More configuration surface area.

### Risks

- **Over-governance.** Universes might lock down too aggressively, reducing agent autonomy. Mitigated by making governance optional---if omitted, no limits apply.
- **Sync conflicts.** Architect and user editing org.yaml simultaneously. Mitigated by git merge semantics and clear ownership (Architect owns state files, users own policy files).

## Alternatives Considered

| Alternative                    | Why rejected                                                          |
| ------------------------------ | --------------------------------------------------------------------- |
| **No org-level config**        | Defaults duplicated across every world and agent. No governance.      |
| **Environment variables only** | Not composable, not versionable, not auditable.                       |
| **Database-backed config**     | Adds dependency, loses human readability, harder to version with git. |
| **Flat config (no cascading)** | Every agent must specify every setting. Verbose and error-prone.      |
