# ADR-019: The product is three spaces: Machine, Agent and OS

**Status:** Accepted
**Date:** 2026-11-22

An agent leaves the status `Proposed`; only the owner writes `Accepted`.

## Context

[ADR-018](018-the-product-describes-itself-as-one-stack-of-six-apps.md) framed Spwn as six apps on one ladder. The owner replaced that structure on 2026-11-22 ("mondes: oui"): the product is organised as three spaces, and the apps sit inside them.

## Decision

Spwn is a stack of apps and abstractions that lets one person build at the scale of a team, and still be the author of what gets built. Its structure is three spaces:

- **Machine**: the premises. The fleet of machines, access, security, documents.
- **Agent**: the workforce. Personalities, roles, focus, responsibilities.
- **OS**: the steering. Flows, projects, vision, products, the system, the abstractions. Wiki, Specs and Design live inside it.

In the app the three spaces are distinct. A switch between them comes first, with Agent and OS greyed until they run.

- **One owner.** The README states the definition and the spaces; the maps of both manuals and the agent brief link to it and none restates it. This keeps ADR-018's rule.
- **Six apps is no longer the structure.** Machines and Agents are the apps that run; Wiki, Specs and Design are being made, inside OS. "OS" names a space, no longer an app.

## Consequences

- Changed: the README's opening and its app section, which becomes the three spaces and their apps; the openings of `CLAUDE.md` and `docs/README.md`. ADR-018 stays as written, marked superseded.
- Left for a later lot: the hero image and its alt text, which still show a tab for each of six apps.
