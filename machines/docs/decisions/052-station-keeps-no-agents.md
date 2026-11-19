# ADR-052: Station keeps no agents; another binary owns them

**Status:** Accepted
**Date:** 2026-11-17

## Context

Station grew a family for coding agents: `agents`, with the contracts `Hooks`, `Journal`,
`Session`, `Multiplexer`, `Forge` and `Repo`, the adapters `claudecode`, `codex`, `herdr`, `gh`,
`git` and `jsonfile`, the `station agents` commands, the `agents.*` calls and
`engine.agents-held`, the `agent-hooks` setting, the guest's report of a VM's journal, the
`agent_*` notices, grants to an agent, and the menu bar's agent page, rows and wait card. It
rests on [ADR-024](024-agents-run-in-herdr.md),
[ADR-025](025-agents-have-their-own-lifecycle.md),
[ADR-027](027-the-agent-hook-keeps-a-scrubbed-line.md) and
[ADR-029](029-an-agent-is-answered-and-merged-from-the-panel.md).
[ADR-045](045-the-agents-family-retires-an-agent-is-a-package.md) proposed retiring it, and
bound station to carrying Claude's own settings in the `claude` package as well.

Asked on 2026-11-17 whether agents should leave the menu bar only or the engine too, the owner
answered: "1 et 2 stp oui, pcque ça va être la responsabilité d'une autre binaire de gérer ça.
Donc commit bien tout stp pour garder la trace des choses au cas où. Mais supprime et simplifie
oui stp".

## Decision

- **Station keeps no notion of a running agent.** The `agents` family, its commands, calls,
  notices, journal files, hook setting and guest report go, and so do the menu bar's and the
  window's agent views, grants to an agent, and every rule of a VM that read an agent: a
  workspace is never idle by its agent's end, and stopping or deleting a VM stops no agent.
- **Another binary owns agents.** Starting, watching, answering and merging an agent is its
  job, outside station.
- **What a machine installs stays a package.** The agent apps (`claude`, `codex`, `herdr`...)
  and the agent skills stay catalog apps, installed, checked and updated as any other.
- **Station keeps what is not an agent's:** a VM's creator (`STATION_SESSION`, the lot that
  made it) and the hold it gives, and grants to a machine.
- **Herdr stays a package, never station's wiring.** Station does not reach a workspace from
  Herdr: the `herdr-machine` setting and the `hc machine` calls go, and a machine that wants
  Herdr carries it by its Machinefile.
- **No compatibility.** Nothing is kept for a client that speaks of agents, and no alias or
  deprecated word stays. The removal lands one commit per unit, each message saying what went
  and why, so git can find and restore it.

## Consequences

- ADR-024, 025, 027 and 029 are superseded by this record, and so is ADR-045, whose
  `claude`-settings half is not decided here.
- The rule of [ADR-049](049-a-vm-signs-itself-in-the-vault-grants-the-rest.md) on an agent's
  grant has no subject left; 049 keeps its own status.
- A VM that still has station's hook in `~/.claude/settings.json` blocks Claude's tool calls
  once `station agents record` no longer exists, so the hook is taken off
  (`station settings off agent-hooks`) before the new binary reaches a VM.
- The other binary starts from nothing in station: it reads no journal, no notice and no call
  of station's.
