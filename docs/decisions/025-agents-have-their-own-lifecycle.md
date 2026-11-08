# ADR-025: Agents have their own lifecycle, apart from their VM

**Status:** Accepted
**Date:** 2026-11-08

## Context

A workspace was made with its agent: `machines create --task --agent` kept the task on the VM's record and started one agent on it once it answered, and every agent verb named the app's session id, read inside the workspace with `--on`. A workspace could not host a second agent the owner could tell apart, an agent could not stop without its VM, and nothing reached an agent without knowing where it ran. The owner's word on DECISIONS A2 (2026-11-08): "ok".

## Decision

- A VM is a machine, an agent is a job on it, and a workspace hosts any number of them, each one Herdr terminal ([ADR-024](024-agents-run-in-herdr.md)) labelled `station.agent=<name>`.
- An agent has a name, short and unique across every machine: a workspace's first agent takes its name, the next ones `<workspace>·<n>`, `--name` gives another. Every verb takes the name alone; on a Mac station reads every running workspace for it. The hook's session id stays the app's own, joined to the agent by its terminal.
- `agents.start` folds a start event into the journal, its one writer's fold, so the agent has its name and task from its first frame; the task lives on the agent, never on the VM record. `machines create` makes the VM only, and `--task` and `--agent` are refused with the `agents start --new` that does it now.
- Stopping an agent (`agents.stop`) closes its terminal and finishes it, its VM untouched; `agents.forget` takes a finished one away. Stopping or deleting a VM stops its live agents first, and its confirmation names them; one that does not stop is a warning, never a refusal.

## Consequences

- `station agents` is a family of the help, no longer plumbing, and `merge-pr` is `merge`.
- `agents.list` and `agents.start` run on a Mac as well as inside a workspace, so every client, the menu bar's as the CLI's, has the one rule: on a Mac `agents.list` reads every machine, or with `On` that workspace's, and `agents.start` (`{Task, Name, App, On | New}`) judges the name against every machine before it makes the throwaway `New` asks or hands the start to the workspace `On` names, which judges it again with what its Mac's engine reads. The other agent calls stay inside the workspace. The list of every machine is each running workspace's read, not yet the engine's report of each VM.
- `machines.cleanup` (another lot) is to offer finished agents and empty throwaways apart, forgetting an agent through `agents.forget`.
