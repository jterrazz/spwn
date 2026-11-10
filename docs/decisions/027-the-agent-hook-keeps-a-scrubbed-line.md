# ADR-027: The agent hook keeps a scrubbed line, never code or commands

**Status:** Accepted
**Date:** 2026-11-08

Written 2026-11-10 from DECISIONS K4 and D-v4 (round 14), SPEC v4 §B5.1 and §D3, and `src/agents/scrub.go`.

## Context

The panel shows what each agent is doing and what it waits on: "review-bot is waiting · PR #42 is ready". Station cannot invent that state (K4), so the `agent-hooks` setting puts a hook into each workspace's Claude Code (`~/.claude/settings.json`) and Codex (`~/.codex/config.toml`), which runs `station agents record` on every event. What an agent says can hold anything: code, a command line, a token, a path outside the project. Whatever the hook keeps goes into the VM's journal, then to its Mac's engine, the menu bar, its notifications and the HUD. SPEC §D3 left the owner one choice: the first line of the last message plus one sentence, relative paths and counts, or the first line only and no paths. The owner's word (D-v4, 2026-11-08): "ok", the recommended option.

## Decision

- The hook keeps two lines of the agent's last message, its **title** (the first line) and its **detail** (at most one sentence after it), each cut at a word to 120 characters (`catalog.AgentLineCap`), and nothing else of what it said.
- A fenced block, or a line whose symbols and keywords outweigh its words, is kept as the words "replied with code" (`catalog.AgentCodeSaid`), never as its text. An inline code span keeps its text, without backticks, only when it reads as a path or a name of 40 characters at most (`catalog.AgentNameCap`). A word shaped like a secret is dropped, by the same rule as the accounts' redaction.
- The agent's task is the first line of its prompt, scrubbed alike. Of the work, the journal keeps steps as a verb, an object (a path relative to the agent's folder, a PR) and counts (lines added and removed, tests passing): a shell command is said by its class alone (`agents.Classify`), never kept, and no diff is.
- `agents.Scrub` runs inside the VM, in the fold of each hook event and of what Herdr says of a terminal, before anything is written to the journal: what it drops never leaves the workspace.

## Consequences

- The menu bar's wait card, Needs you's wait row and the `agent_waiting` notice speak in the agent's own first words, with no code in them.
- A long answer, or one that is mostly code, reads as "replied with code"; the owner opens the agent's terminal (Attach) to see the rest.
- The stricter alternative (the first line only, no paths) is one change to `Scrub` and its suite, `src/agents/scrub_test.go`.
