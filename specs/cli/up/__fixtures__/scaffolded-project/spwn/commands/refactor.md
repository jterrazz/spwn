---
name: refactor
description: Refactor the selected code while preserving behaviour.
---

# Refactor

Refactor the code I've selected (or the file I'm in if nothing is
selected) without changing observable behaviour. Prioritise:

1. **Smaller, named functions** when blocks of logic do one thing.
2. **Earlier returns** instead of nested conditionals.
3. **Removing dead code** that nothing references.
4. **Naming** — variables and functions should describe intent, not type.
5. **Tests stay green** — run them after each non-trivial change.

Do not introduce new dependencies or change public APIs unless I ask
for it. When you spot a refactor that would be valuable but is out of
scope, mention it at the end as a follow-up rather than doing it.
