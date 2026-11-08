# ADR-023: Agents run in Herdr, and tmux is no adapter

**Status:** Accepted
**Date:** 2026-11-08

## Context

The agents' `Session` contract had one adapter, `tmux`: every agent of a workspace ran in its tmux session `main`, a reply was pasted into its current pane, and nothing told one agent's pane from another's. The owner works in Herdr: herdr-core (`hc`) runs every terminal and agent and keeps them alive, herdr-app shows them, and a machine added with `hc machine add` shows its terminals beside the Mac's own. The owner's word (2026-11-08): only Herdr, the tmux adapters gone for now.

## Decision

- `Session` has one adapter, `agents/adapter/herdr`. An agent starts in a terminal of its own on the workspace's herdr-core, labelled `station=agent` and `station.app=<app>`, and its task is typed once the agent is ready. Every call is one request of herdr-core's API on `hc connect`'s stdin, so a text typed to an agent is never on a command line.
- An agent's terminal is kept on its journal entry: the hook runs inside it and hands `HC_TERMINAL_ID` to `agents.record`. A reply goes to that terminal; an agent no terminal holds is refused (`agents.no-terminal`).
- Herdr's own status of a terminal (working, blocked, idle, done) says the agents no hook reports. The hooks still win: they carry the message and the steps, which Herdr does not.
- A new contract, `Multiplexer`, is the Mac's side: the machines its Herdr reaches, and attach. A workspace made on a Mac is added to that Mac's Herdr (`hc machine add --install`), forgotten when deleted; the setting `herdr-machine` checks every workspace is reached, and applies what is missing. `station agents attach <session> --on <vm>` runs `hc attach <vm>/<terminal>`.
- `gh` (the merge) and `git` (the shortstat) stay: they are not session adapters.

## Consequences

- No agent station starts runs in tmux any more. The owner's shell is untouched: the tmux app, its dotfiles, and the `jj`, `jc`, `jo` and `jg` shortcuts, which still open Claude, Codex and Gemini in tmux by hand ([dotfiles](../reference/dotfiles.md)).
- A workspace of another Mac is reached by that Mac's Herdr, not by the owner's: attaching to it from the owner's Mac is not built.
