---
name: architecture
description: Use when designing a new system, component, or significant refactor — trade-offs, boundaries, invariants.
---

# Architecture

Use this skill when designing a new system, component, or significant
refactor.

## Instructions

1. Define the boundaries: what is inside this system, what is outside,
   and what are the contracts at each edge.
2. List the data flows -- where data enters, how it transforms, where
   it lands. Draw this as a simple text diagram if it helps.
3. Identify the hardest constraint (latency, consistency, team size,
   deployment target) and design around it first.
4. Choose boring technology by default. Only introduce a new dependency
   if the boring option fails a stated requirement.
5. Write the design as: context, decision, consequences. Keep it under
   one page. If it needs more, the system is too big -- split it.
6. Include a "what could go wrong" section with at least two failure
   scenarios and their mitigations.
