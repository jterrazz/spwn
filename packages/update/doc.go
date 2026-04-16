// Package update powers `spwn upgrade` and the CLI's version-check
// banner.
//
// Two entry points:
//
//   - CheckForUpdate(ctx, client, current, CheckOpts) → *Plan —
//     queries GitHub releases (stable or beta channel), resolves
//     the right platform asset, returns a Plan containing
//     Release/Asset/Platform/Latest/UpdateAvail without
//     side-effects.
//   - Apply(ctx, plan, ApplyOpts) — downloads the binary, verifies
//     its SHA256 against the release's SHA256SUMS, and atomically
//     replaces the current executable.
//
// CheckLatestVersion + GetVersionInfo are lightweight variants
// used by the startup banner (`spwn --version` hint) and the web
// UI's "update available" badge. CLIVersion is the var that
// -ldflags -X "=" populates at build time.
package update
