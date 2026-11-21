# ADR-018: The product describes itself as one stack of six apps

**Status:** Proposed
**Date:** 2026-11-21

An agent leaves the status `Proposed`; only the owner writes `Accepted`.

## Context

Four definitions of Spwn grew side by side in this repository and around it: "the control plane for AI agents" (the web UI's welcome), "the operating system for autonomous agent worlds" (the domain model and the agent brief), "a framework, not a product" (ADR-010's "framework identity") and "the set of abstractions, rules and controls" (the README's hero). The help of the `spwn` binary still spoke of artificial life: "create realities for things that can think", "a living identity that inhabits a world", "the Big Bang".

The owner redirected the framing on 2026-11-17: "non la vision globale c'est vraiment stack, ensemble d'applis et s'absatractions qui te permettent d'etre toi x 100". On 2026-11-21 he accepted the brand's vision brief, its definition and the order of its six apps among it ("tout ok pour [the brief]"), and gave this lot its go: "tout ok go les 4".

"Operating system" now also clashes with one of the six apps, which is called OS.

## Decision

Spwn describes itself with one definition: a stack of apps and abstractions that lets one person build at the scale of a team, and still be the author of what gets built. It is six apps on one ladder: Machines and Agents run, Wiki, Specs, Design and OS are being made.

- **One owner.** The README states the definition and the six apps. The maps of both manuals name it (a map links only to its own pages), and the domain model and the agent brief link to it. None restates it.
- **Each app says what it is.** Agents is the app that runs one agent, or a team with a chief and workers, in a world you declare, in any harness; its manual says so. Machines' manual names Machines and its command, `station`.
- **The help says what a command does.** A Cobra `Short` or `Long` is a plain statement of the command's effect, with no framing.

## Consequences

- Changed: the README's opening and the order of its app table; the opening of `docs/README.md`, `docs/concepts/domain-model.md`, `machines/docs/README.md` and `CLAUDE.md`; the `Short` and `Long` of `spwn`, `spwn agent`, `spwn architect`, `spwn up`, `spwn world up` and `spwn agent sleep`, with `docs/reference/commands.md` and the `up` usage spec; the web UI's welcome line. The description of Sleep now matches its code in both pages.
- The site copies the generated command reference (`content/cli/` in spwn-web). It must be refreshed at the next release, or it keeps the old help.
- Left for a later lot: the hero image, which still shows the old headline, the old definition and a tab named Station; the agent's own prompt (`packages/agent/internal/mind/mind.go`, "living inside an isolated world") and the CLI skill's "the universe", which are behaviour and golden fixtures; the output line "Agent is alive."; Machines' output words that say Station; the titles of past decisions, which stay as written.
