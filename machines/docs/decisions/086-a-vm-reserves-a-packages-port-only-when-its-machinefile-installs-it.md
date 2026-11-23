# ADR-086: A VM reserves a package's port only when its Machinefile installs it

**Status:** Accepted
**Date:** 2026-11-23

Accepted by the owner on 2026-11-23, on the owner's word.

Amends [ADR-035](035-a-port-is-a-right-with-an-audience.md).

## Context

ADR-035 moved the dev-server profiles into the catalog: a package's `ports` names the port it serves on and the env that carries its URL (`vite` 5173, `expo` 8081), and the engine reserves every such port, inside the VM's rights, at each run of the VM, so the env holds before the server starts. It reserves them for every VM, whatever the VM carries. A workspace made from `base-ubuntu`, which installs neither Vite nor Expo, so holds two reserved ports of its Mac and their two variables, and the owner's ports list shows two shares nothing will ever serve. The owner decided on 2026-11-23 that a VM reserves and shares those ports only when its Machinefile installs the package that serves them.

## Decision

At each run of a VM, the engine reserves a port a package's `ports` sets an env for, and renders that env, only when the VM's Machinefile installs that package, inside its rights as before. What a Machinefile installs is its plan's packages, resolved for the VM as its own apply resolves it: those it lists, those it bakes, and those they bring in by what they require. The port belongs to the package alone: its catalog entry declares it, and the engine holds no list of dev-server ports, so a package the owner writes with `ports` and `env` is reserved by the same rule.

The Machinefile is the one the rights are read from (the one named after the VM, else the one it was made from), read at the same moment: each run of the VM. A VM with no Machinefile installs nothing and so reserves no such port; neither does one whose Machinefile cannot be read or whose plan cannot be resolved, which the engine's log says.

## Consequences

- A VM whose Machinefile comes to install Vite or Expo gets its port and env at its next run (create, `start`, `restart`, its own run at boot), as a rights file edited does; an `apply` inside the VM installs the package but reserves nothing until then.
- A VM whose Machinefile no longer installs such a package reserves nothing for it at its next run. A reservation an earlier run made stays, as any port its rights reserved does, until its idle days, the VM's delete or `ports stop-sharing` release it: the engine does not tell a reservation made for an env from one made for a listener.
- A port a package serves with no env (Node's 3000, Postgres's 5432) is unchanged: it is reserved when something listens there, inside the rights, whatever the Machinefile installs. Its label ("probably Vite") still comes from every package of the catalog, installed or not: a hint, never a decision.
- A package file or Machinefile the decoder refuses, anywhere in the sources, leaves every plan unresolved, as `plan` and `explain` refuse it, so a run in that state reserves no package port; the rights, read from the VM's own chain alone, still hold.
