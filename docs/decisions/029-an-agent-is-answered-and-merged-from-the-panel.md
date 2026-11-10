# ADR-029: An agent is answered and merged from the panel

**Status:** Accepted
**Date:** 2026-11-08

Written 2026-11-10 from DECISIONS D-v4 (round 14), SPEC v4 §D3, and `src/pages/menubar/agent.go`.

## Context

An agent that waits on the owner waits for a reply, or for its pull request to be merged. Both act inside the agent's world: a reply is text typed into its Herdr terminal ([ADR-024](024-agents-run-in-herdr.md)), a merge is `gh pr merge` with the GitHub account lent to its workspace. The prototype does both from the panel, Merge without a confirmation. SPEC §D3 offered the owner that, or a Merge that asks y/n. The owner's word (D-v4, 2026-11-08): "ok", the recommended option.

## Decision

- A waiting agent's page carries Reply (Return opens its one field, which `agents.reply` types into the agent's terminal) and, once it opened a pull request, Merge, labelled with the PR it merges (`Merge PR #42`), which runs `agents.merge` at once. Neither asks for a confirmation; the HUD says what was done.
- Stop keeps the alert its graded confirmation level gives it ("Stop the agent review-bot?"), as every verb that ends something does; Reply and Merge name in their label exactly what they act on.

## Consequences

- The owner answers an agent and lands its work without opening a terminal.
- A Merge pressed by mistake is a merged pull request: the safeguard is the button's words, not a dialog.
