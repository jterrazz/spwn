# packages/update

CLI self-update + version-check.

## Role

Owns `spwn upgrade`: fetches release metadata from GitHub, downloads the platform-appropriate binary, verifies it against the SHA256SUMS published with the release, and atomically replaces the current binary. Also exposes a lightweight version-check used by the CLI startup banner and the web UI's "update available" badge. Split from the former `packages/upgrade` so self-update and state migration (`packages/migration`) are independently composable.

## Key types

- `CLIVersion` — set at build time via `-ldflags`, defaults to `"dev"`. The CLI entrypoint propagates its own `Version` into this var at startup.
- `CheckLatestVersion(maxAge)` — tag string from GitHub, file-cached under `~/.spwn/.version-check`. Returns `""` on any error so the caller can fall back silently.
- `VersionInfo` / `GetVersionInfo(maxAge)` — structured current/latest/update-available/release-url.
- `Version` — parsed semver type (`Major.Minor.Patch[-prerelease]`) + `ParseVersion(s)`.
- `GitHubClient` / `CheckForUpdate(ctx, client, current, opts)` → `Plan` — the release-fetch + planning surface.
- `Apply(ctx, plan, opts)` — downloads, verifies, and atomically installs.

## Related

- **Imported by** — `apps/cli` (`spwn upgrade`, startup version banner), `apps/api` (web UI version badge)
- **Imports** — `packages/platform` (for `~/.spwn/` cache path)
