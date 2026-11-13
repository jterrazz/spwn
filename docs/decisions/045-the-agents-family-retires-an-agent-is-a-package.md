# ADR-045: The agents family retires, and an agent is a package

**Status:** Proposed
**Date:** 2026-11-13

## Context

Station grew a family of its own for coding agents: `agents`, with the contracts `Hooks`, `Journal`, `Session`, `Multiplexer`, `Forge` and `Repo`, the `station agents` commands, the menu bar's agent views and one journal file per session under `{state}/station/agents`. It rests on four records: agents run in Herdr and tmux is no adapter ([ADR-024](024-agents-run-in-herdr.md)), an agent has its own lifecycle apart from its VM ([ADR-025](025-agents-have-their-own-lifecycle.md)), its hook keeps a scrubbed line ([ADR-027](027-the-agent-hook-keeps-a-scrubbed-line.md)) and it is answered and merged from the panel ([ADR-029](029-an-agent-is-answered-and-merged-from-the-panel.md)).

What station is for has moved since: it sets up and runs a Mac, its VMs and its servers from a catalog of packages ([ADR-031](031-the-catalog-defines-the-machinefile-chooses.md)). Running agents, supervising them and merging their work is a product of its own, and every agent tool is already something the catalog can put on a machine: Herdr and Claude are packages in the category `agent`. The design round of 2026-11-12 (section G) gave the `claude` package its own settings, on every Mac where Claude is installed.

## Decision

- **The `agents` family retires.** Its contracts, its adapters (`claudecode`, `codex`, `jsonfile`, `herdr`, `gh`, `git`), `station agents`, the `agent_started`, `agent_waiting` and `agent_finished` notices, the guest job that reports a VM's agents, the menu bar's agent views and the journal files go. Station keeps no notion of a running agent.
- **An agent is a package.** `herdr`, `claude` and the like stay catalog packages of the category `agent`, installed, checked and updated as every other package is. The `claude` package carries what its own settings need (`cleanupPeriodDays`, the status line, the PreToolUse hook and the two scripts it runs), versioned in the package.
- **The four records are replaced.** ADR-024, 025, 027 and 029 are superseded by this one once it is Accepted; their statuses move then, not before, so the graph closes both ways in one commit. This record deletes nothing itself: the code goes in a lot of its own.

## Consequences

- A later lot deletes the family and what reads it: the contracts and adapters, the wire and menu bar surfaces, the specs and stories that exercise them, and the pages of the manual that describe them.
- Station no longer answers an agent's question or merges its work; a tool that does is a package the owner installs, outside station.
- The scrubbed-line rule of ADR-027 has no subject left: nothing of station reads an agent's hook.
- The `claude` package takes the settings the family's `agent-hooks` setting wrote, so a Mac keeps them where Claude is installed.
