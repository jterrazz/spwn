---
name: studio-stack
description: 'Overview of the @jterrazz ecosystem: shared npm packages, project naming ({product}-{role}, package-*), required files, CI/CD workflows, testing and architecture conventions. Use when working on any jterrazz project, choosing a package, scaffolding decisions, or understanding how the stack composes.'
---

# @jterrazz Stack

The map of the `@jterrazz` ecosystem: composable packages, one set of conventions, interchangeable repos. Canonical home: `studio-station` `docs/concepts/stack.md`.

## Mental model

- **Shared packages, one concern each**: `@jterrazz/typescript` (toolchain) and `@jterrazz/test` (testing) under every project; `telemetry`, `intelligence`, `analytics`, `broadcast`, `manifest`, `attestation` where a project needs them. CI/CD is shared workflows from `studio-actions`; infra is `studio-infrastructure`.
- **Two project shapes**: libraries (`package-*`, published `@jterrazz/*`, ports & adapters) and applications (`{product}-{role}`, hexagonal). Both build/lint/test through `@jterrazz/typescript` and `@jterrazz/test`.
- **A project declares what it is once**: `studio.config.ts` (kind, platform, presets, exceptions with reasons); the editor files are one-line derivations, the Makefile answers the six gestures (`install`, `build`, `lint`, `fix`, `test`, `check`), the validate workflow is shared, and the root files are the repo-structure doctrine's.

## Where to look

| Task                                                | Read                                      |
| --------------------------------------------------- | ----------------------------------------- |
| The whole picture: packages, naming, required files | `studio-station` `docs/concepts/stack.md` |
| Build / lint / format / API docs                    | `studio-typescript` skill                 |
| Testing conventions and structure                   | `studio-test` skill                       |
| Scaffolding a new project                           | `studio-new-project` skill                |
| CI/CD workflows                                     | `studio-workflows` skill                  |
| Where knowledge lives in a repo                     | `studio-repo-structure` skill             |
| What is installed on the machine                    | `studio-toolbelt` skill                   |

## Always

- `npm` only (never pnpm/yarn), Node.js 24, ESM only, `.js` extensions in Node imports.
- Author: `Jean-Baptiste Terrazzoni <contact@jterrazz.com>`.
- Never restate another skill's territory here; route to it.
