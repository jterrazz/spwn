# ADR-014: The code is families, each a hexagon

**Status:** Proposed
**Date:** 2026-11-06

Written 2026-11-06 from the owner's round 13 decisions (Q2, E1, E2) and the code that carries them.

## Context

The v2 tree was layers: input, command, domain, presenter, output, with a `system` door the domain reached the machine through. A reader opening `src/` saw the plumbing first, and a feature lived in five folders. Every external program was reached through one door, so replacing tart, launchd or Tailscale meant reading the whole domain for the places that assumed it, and the laptop and the desktop Mac were told apart by model name. The owner asked for the business first and the plumbing plainly named: each capability its ports, its adapters, its strategies and its applicability rule.

## Decision

The top of `src/` names the families (`machines`, `apps`, `settings`, `disk`, `ports`, `accounts`, `background`, `awake`, `activity`, `images`, `overview`, `setup`, `engine`). Inside each, four parts, always the same names: the root package is the family's pure model and its rules, `contract/` its contracts (one interface per need, with `contracttest/`, the suite every adapter and the fake pass), `calls/` its use cases (`<family>.<verb>`, each a `call.Call` with its input, output, its `Where` and its `Applies`), and `adapter/<tech>/` one adapter per technology, `adapter/fake/` over the fake world. A contract with several adapters chosen per item is a strategy, through one registry in its contract package (`Installers`, `Appliers`, `PublicRoutes`); no other code switches on a method, an effect or a route. The hypervisor is one contract, `Hypervisor`, with `tart` its one real adapter.

What a machine can do is its `core/machine.Profile`: its kind, its system and the capabilities its adapters report (a battery, a lid, a job manager, a hypervisor). A rule reads capabilities, never a model name, and `core/section.For(kind)` is the one table of which kind has which section. Every surface reaches a call through the `core/call.Router`, which decides where it runs ([ADR-016](016-api-v2-and-remote-by-api.md)). The architecture test (`specs/architecture/`) holds the tree, the imports, the technology words, the contracts' completeness and the gate of every call.

This supersedes [ADR-001](001-station-mirrors-the-studio-cli-layers.md) (the layers) and [ADR-006](006-tart-is-the-engine-behind-a-port.md) (tart behind the `Engine` port, now the one real adapter of the `Hypervisor` contract).

## Consequences

- A new hypervisor, job manager, tunnel, secret store or notifier is one folder under its family's `adapter/` and one line in a registry or in `wire`; the use cases are unchanged and its suite of `contracttest` says whether it is whole.
- A call cannot run a command or open a file: only adapters import `platform/local`, so every use case is testable over the fake world ([ADR-020](020-the-fake-world.md)).
- What ADR-001 kept still holds: one error block, exit codes from returned errors, Fang behind the CLI's own error layer, one palette, the tab-separated pipe.
- The allowlists of the architecture rules only shrink; a family that needs a new exception stops and asks.
