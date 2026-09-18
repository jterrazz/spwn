---
name: mempalace
description: Use when reading from or writing to the agent's persistent memory palace — cross-session notes, facts, and playbooks.
---

# mempalace — your memory palace

You have a persistent memory palace. It lives on your agent's host and
survives across sessions — when a world is rebuilt, the knowledge you
saved yesterday is still here. The **mempalace** CLI is available on
your PATH; shell out to it whenever you need to recall or record
something long-lived.

## What it is

MemPalace is a local, raw-verbatim memory store. It organizes
knowledge into **wings** (people, projects), **halls** (types of
memory), and **rooms** (specific ideas). You don't have to remember
the layout — search by meaning.

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

Discover the current CLI surface with `mempalace --help`; the exact
flags evolve, but the core verbs are stable. Run it as a subprocess
from your shell.

Typical flow:

1. Before answering a question you might know the answer to, search
   first. Pass a natural-language query to the search verb. If it
   returns verbatim context, use it.
2. After completing meaningful work, write it down with the save
   verb and a short summary of the fact to keep.
3. When the user tells you something important about themselves or
   the project, save it immediately — don't wait for a "remember
   this" instruction.

If a call fails or the CLI prints a new banner you didn't expect, rerun
`mempalace --help` to re-learn the surface before retrying.

## What NOT to do

- Don't treat the palace as a dumping ground. Paraphrased one-liners
  beat pasted walls of text.
- Don't overwrite — add. Contradictions are data; deletions are lossy.
- Don't save secrets, credentials, or API keys. The palace is local
  but it's still on disk.

## One-line mantra

**Search before you answer. Save before you forget.**
