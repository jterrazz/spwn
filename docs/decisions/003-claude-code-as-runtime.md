# ADR-003: Claude Code CLI as Agent Runtime

**Date:** 2026-02-28
**Status:** Accepted

## Context

Spwn needs an agent runtime—the component that reads the Mind, understands the Physics, and autonomously operates inside a universe. The original design assumed we'd build custom AI logic: a TypeScript SDK wrapping LLM API calls with tool-use loops, context management, and session handling.

But Claude Code already mastered this. It reads markdown files natively. It has `bash`. It composes Unix commands. It manages its own context window. It persists sessions. It's exactly the runtime we'd spend months building—and it already works.

The question became: why build another runtime when the best one already exists?

## Decision

Use Claude Code CLI as the agent runtime spawned inside universes. Spwn's job is **reality infrastructure**—creating universes, managing agents (their Minds, physics, faculties), defining physics, bridging faculties. Claude Code's job is **thinking and acting** inside that reality.

The architecture:

1. **Architect** creates a universe (Docker container)
2. **Architect** mounts the Mind at `/mind` and workspace at `/workspace`
3. **Architect** generates `physics.md` describing available tools and faculties
4. **Container-side Gate** spawns Claude Code CLI via ACP (Agent Client Protocol)
5. **Claude Code** reads the Mind (personas, skills, knowledge, playbooks) as markdown—its native format
6. **Claude Code** reads `physics.md` to understand the environment
7. **Claude Code** operates autonomously using bash, files, and all available tools
8. **Claude Code** writes to the journal as it works
9. **Spwn** manages session persistence across container lifecycle

### Session Persistence

The container-side Gate manages sessions via ACP's `session/load` and `session/new`. Spwn manages session IDs so that an agent re-entering a universe (or entering a new one with the same Mind) can continue from where it left off. See [the ACP adoption decision](./006-acp-inside-container.md).

### Mind ↔ Claude Code Mapping

The agent profile directory structure maps naturally to Claude Code's conventions:

| Mind Directory | Claude Code Equivalent         | How Claude Code Uses It            |
| -------------- | ------------------------------ | ---------------------------------- |
| `personas/`    | `.claude/agents/`              | System-level identity and behavior |
| `skills/`      | `.claude/skills/`              | Invocable capabilities             |
| `knowledge/`   | `CLAUDE.md`                    | Context and facts                  |
| `playbooks/`   | `.claude/skills/` (procedural) | Step-by-step procedures            |
| `journal/`     | —(Spwn-specific)               | Append-only experience log         |

## Consequences

### What Becomes Easier

- **No custom agent logic.** We don't build, debug, or maintain an AI reasoning engine.
- **Instant capability.** Claude Code already handles tool use, error recovery, multi-step planning, and context management.
- **Native markdown consumption.** Mind files are markdown—Claude Code reads them without any custom loader.
- **Session management.** Claude Code's `--resume` provides session persistence out of the box.
- **Ecosystem alignment.** Claude Code is actively maintained by Anthropic. We ride their improvements for free.

### What Becomes Harder

- **Vendor coupling.** Spwn depends on Claude Code CLI being available and maintained.
- **Multi-model support.** Supporting other CLI agents (Codex CLI, Gemini CLI) needed an abstraction layer; [the ACP adoption decision](./006-acp-inside-container.md) supplied it, so swapping agent CLIs is a container image change.
- **Fine-grained control.** We can't control Claude Code's internal reasoning—only what we put in the environment.
- **Licensing.** Claude Code's license terms apply inside universes.

## Alternatives Considered

| Alternative                      | Why Rejected                                                                                            |
| -------------------------------- | ------------------------------------------------------------------------------------------------------- |
| **Custom TypeScript agent SDK**  | Months of work to build what Claude Code already does. We'd be a worse version of it.                   |
| **Raw LLM API calls**            | No session management, no tool use loop, no context management. We'd rebuild Claude Code from scratch.  |
| **LangChain/CrewAI integration** | Tool-call frameworks—the exact paradigm Spwn replaces. They chain functions; we give agents a computer. |
| **Model-agnostic agent loop**    | Attractive in theory but Claude Code's Unix mastery is unmatched. Build for the best, abstract later.   |
