# The stack

How every `@jterrazz` project composes: the shared packages, the naming scheme, the required files, and the conventions that make repositories interchangeable. Deep knowledge stays in each package's own manual; this page is the map of the whole.

## Shared packages

| Package                  | Owns                                                                                  | Manual                       |
| ------------------------ | ------------------------------------------------------------------------------------- | ---------------------------- |
| `@jterrazz/typescript`   | The toolchain: the settings file, build, quality checks, lint presets, mutation       | `package-typescript/docs/`   |
| `@jterrazz/test`         | Testing: the specifications, the conventions and where a test lives, the doubles      | `package-test/docs/`         |
| `@jterrazz/telemetry`    | Logs, traces and metrics over OpenTelemetry, behind ports                             | `package-telemetry/docs/`    |
| `@jterrazz/intelligence` | The AI SDK composition layer: providers, cost, fallback, logging middleware           | `package-intelligence/docs/` |
| `@jterrazz/analytics`    | Product analytics behind a port: typed events, the OpenPanel adapter                  | `package-analytics/docs/`    |
| `@jterrazz/broadcast`    | Multi-channel announcements: App Store in-app events, push notifications              | `package-broadcast/docs/`    |
| `@jterrazz/manifest`     | A site's machine-facing presence declared once: metadata, JSON-LD, sitemap, its audit | `package-manifest/docs/`     |
| `@jterrazz/attestation`  | Article attestation: an EIP-712 signature anchored to Bitcoin                         | `package-attestation/docs/`  |

Shared CI/CD lives in `jterrazz/studio-actions` (`check`, `deploy-docker`, `publish-npm`, `publish-go`); infrastructure in `jterrazz/studio-infrastructure` (K3s, Helm, Traefik).

## Project types and naming

```text
{product}-{role}             # applications: signews-api, signews-web, signews-mobile, ...
package-{name}               # shared libraries, published as @jterrazz/*
```

Roles: `-api`, `-web`, `-mobile`, `-broadcast`, `-blueprint`, `-landing`.

A project declares what it is once, in `studio.config.ts`: its `kind` (`library`, `api`, `cli`, `website`, `mobile` or `workspace`) and its `platform`, from which the toolchain derives the rest. The two lists, and what each kind and platform imply, are `@jterrazz/typescript`'s [Configuration](https://github.com/jterrazz/package-typescript/blob/main/docs/reference/configuration.md).

## Required files

Every project carries:

- `studio.config.ts`: the kind, the platform, the presets it is held to, and every exception with its reason.
- The editor files the toolchain derives and `typescript fix` writes (`tsconfig.json`, `oxlint.config.ts`, `oxfmt.config.ts`, and `vitest.config.ts` from `@jterrazz/test`'s preset), each one line. No other program keeps its settings file at the root.
- A `Makefile` answering the six gestures (`install`, `build`, `lint`, `fix`, `test`, `check`), with `check` the whole gate.
- `.github/workflows/check.yaml` calling the shared workflow, which runs `make build`, `make lint`, `make test`, then `make mutation` where the Makefile has one.
- The root files mandated by the [repo-structure doctrine](repo-structure.md): the README front door, `CLAUDE.md`, a `docs/` manual cut by mood.

How each is written is `@jterrazz/typescript`'s [Getting started](https://github.com/jterrazz/package-typescript/blob/main/docs/start/getting-started.md); scaffolding one from nothing is the `studio-new-project` skill.

A repository whose product is not TypeScript carries them all the same, for the half that is: `station` is a Go binary whose end-to-end harness under `specs/` declares itself a `cli` on `node` held to `@jterrazz/test`'s preset. Its own language answers to its own linter (`.golangci.yml`, at the version the Makefile pins), and each gesture runs both halves. Its menu bar app, `apps/menubar`, is a Swift package with no Xcode project: `make check` ends with its `swift test` and its bundle.

## Testing

The convention is `@jterrazz/test`'s, and every project follows it: the SUBJECT decides where a test lives. A module test proves one unit from the `__tests__/` folder of its module's folder (`dir/__tests__/foo.test.ts` for `dir/foo.ts`); the assembled product is specified from outside, under `specs/<kind>/`, through the runner its `<kind>.specification.ts` builds, and a terminal session is a `<case>.spec.yaml` document. What a spec stands on sits in the closed list of special folders (`__setup__/`, `__fixtures__/`, `__expected__/`, `__requests__/`, `__seeds__/`) at the lowest folder its readers share. The full convention is `package-test`'s manual: route there, don't restate it.

## Architecture

Libraries use **ports & adapters** (`src/ports/` interfaces, `src/adapters/` implementations, `src/index.ts` the entry). Applications are **hexagonal** (`src/domain/`, `src/application/`, `src/infrastructure/`), built in one file, the composition root `src/composition/compose.ts`, with no DI container. The layer map is `@jterrazz/typescript`'s [hexagonal map](https://github.com/jterrazz/package-typescript/blob/main/docs/reference/quality-checks.md#the-hexagonal-map), the law and [The composition root](https://github.com/jterrazz/package-typescript/blob/main/docs/concepts/the-composition-root.md).

## Standing conventions

- `npm` only, never pnpm or yarn; every repo commits `package-lock.json`. A unit on the `bun` platform is the one exception, and declares it.
- Node.js 24, ESM only (`"type": "module"`), `.js` extensions in Node imports.
- Author: `Jean-Baptiste Terrazzoni <contact@jterrazz.com>`.

## Related

- [Repo structure](repo-structure.md): where knowledge lives in every repo.
- [Apps and skills](../reference/apps-and-skills.md): the machine-side registries that distribute the skills.
