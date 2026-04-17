# packages/dependency

Parser for `tool.yaml` dependency manifests + the project lockfile.

## Role

Every spwn dependency — whether shipped in the built-in catalog or authored under `spwn/tools/<name>/` in a user project — is described by a single `tool.yaml`. This package is the shared parser: it reads one of those manifests via an abstract `Resolver` (host filesystem or `go:embed`), turns it into a typed `Schema`, and classifies the ref (`spwn:<name>`, `github.com/...`, or local bare name). It also owns `spwn.lock`, the line-oriented text file pinning each dep's version. No Docker, no image building — just parsing + ref classification.

## Key types

- `Schema` — on-disk shape of `tool.yaml`: name, kind, version, dependencies, install spec, files, verify, runtime-config injection.
- `Parse(Resolver, ParseOptions) → *Parsed` — the single entry point. `DirResolver` reads from disk; `EmbedResolver` reads from `embed.FS`.
- `Ref`, `ParseRef(string) → Ref` — classify a ref as `KindLocal`, `KindSpwnBuiltin`, or `KindRegistry`.
- `SplitVersion`, `Canonical` — strip and normalise `@version` suffixes.
- `ResolveTool` / `ResolveSkill` — check whether a ref resolves to a real on-disk target.
- `Lockfile` + `LoadLockfile` / `SaveLockfile` — read/write `spwn.lock` (`<ref> <version> <source>` per line).

## Related

- **Imported by** — `apps/cli`, `catalog`, `packages/architect`, `packages/image` (adapter), `packages/project` (validator), `packages/runtimes`
- **Imports** — stdlib only
