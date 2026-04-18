---
name: mempalace
description: Use when reading from or writing to the agent's persistent memory palace — cross-session notes, facts, and playbooks.
---

# mempalace — your memory palace

You have a persistent memory palace. It lives on your agent's host and
survives across sessions — when a world is rebuilt, the knowledge you
saved yesterday is still here. That memory is exposed via the
**mempalace** MCP server, which this agent was spawned with.

## What it is

MemPalace is a local, raw-verbatim memory store. It organizes
knowledge into **wings** (people, projects), **halls** (types of
memory), and **rooms** (specific ideas). You don't have to remember
the layout — the MCP tools let you search by meaning.

Nothing leaves the machine. No API keys, no cloud round-trips. Every
recall is semantic search over what you've already written down.

## When to use it

Reach for mempalace whenever a piece of information should outlive the
current turn:

- A decision and the reasoning behind it ("we switched to Postgres
  because the Mongo joins got ugly").
- A gotcha you solved ("the `make build` target needs `GOOS=linux` on
  darwin hosts").
- A fact about another agent, a project, or a user that changes how
  you'll act next time.
- The shape of a system you just learned so the next session doesn't
  have to rediscover it.

Do NOT use mempalace for ephemeral chatter or single-turn scratch
work — that clutters the palace and dilutes search quality.

## How to call it

The MCP server is already wired into your runtime config. The tools
appear under the `mempalace_*` prefix. You don't need to invoke a CLI
or restart anything — the runtime loads the server on boot.

Typical flow:

1. Before answering a question you might know the answer to, search:
   call `mempalace_search` with a natural-language query. If it
   returns verbatim context, use it.
2. After completing meaningful work, write it down: call
   `mempalace_save` (or the equivalent "add a memory" tool exposed
   by the server) with the fact you want to keep.
3. When the user tells you something important about themselves or
   the project, save it immediately — don't wait for a "remember
   this" instruction.

If you're ever uncertain which tool to call, list the mempalace tools
first and pick the closest match. The server's tool descriptions are
the source of truth.

## What NOT to do

- Don't treat the palace as a dumping ground. Paraphrased one-liners
  beat pasted walls of text.
- Don't overwrite — add. Contradictions are data; deletions are lossy.
- Don't save secrets, credentials, or API keys. The palace is local
  but it's still on disk.

## One-line mantra

**Search before you answer. Save before you forget.**
