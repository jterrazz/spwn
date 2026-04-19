# Update System

This document describes how spwn ships and updates itself. The goal is a
**GitHub-only** distribution channel: no custom servers, no external
infrastructure. Every artifact lives in a GitHub Release.

---

## Artifacts

Two things are distributed:

| Artifact              | Channel                 | Auto-update mechanism                     |
| --------------------- | ----------------------- | ----------------------------------------- |
| `spwn` CLI binary     | GitHub Releases (tar.gz) | `spwn upgrade` + background version check |
| spwn spwn.app  | GitHub Releases (bundles)| Tauri updater plugin → `latest.json`      |

Both are built from the same git tag (`vX.Y.Z`). A single `git push --tags`
triggers the entire pipeline.

---

## CLI auto-update

### Release flow (CI)

`.github/workflows/release.yaml` triggers on any tag matching `v*`. It runs
**GoReleaser** (`.goreleaser.yml`) which:

1. Cross-compiles `spwn` for `darwin_{arm64,amd64}` and `linux_{arm64,amd64}`.
2. Embeds the git tag into the binary via `-ldflags "-X spwn.sh/apps/cli.Version={{.Version}}"`.
3. Dependencies each binary into `spwn_{os}_{arch}.tar.gz`.
4. Computes SHA256 of every archive into `checksums.txt`.
5. Uploads archives + `checksums.txt` to the GitHub Release.

### Client flow (`spwn upgrade`)

Implemented in `packages/base/update/`. High-level sequence:

```
┌─ CheckForUpdate ─────────────────────────────────┐
│ GET api.github.com/repos/jterrazz/spwn/releases/ │
│     latest  OR  ?per_page=10 (for --channel beta)│
│ → parse, compare against embedded Version        │
│ → find asset matching GOOS_GOARCH                │
│ → find checksums.txt in the same release         │
└──────────────────────────────────────────────────┘
                  │
                  ▼
┌─ Apply ──────────────────────────────────────────┐
│ 1. Stop running worlds + architect (best-effort) │
│ 2. Download asset into tmpdir (3 retries)        │
│ 3. Download checksums.txt                        │
│ 4. SHA256 the archive, match against digest in   │
│    checksums.txt - refuse install on mismatch    │
│ 5. Extract binary from the tar.gz (Go stdlib)    │
│ 6. os.Rename new binary → target path (atomic)   │
└──────────────────────────────────────────────────┘
```

**Security properties:**

- **Verified integrity.** Every binary is checked against a SHA256 that
  sits next to it in the same release. An attacker would need to replace
  both files in the release AND control the API response, which requires
  a compromised GitHub account rather than just a TLS cert.
- **Atomic install.** POSIX uses `os.Rename()`; the kernel guarantees that
  readers see either the old inode or the new inode, never a partial
  write. Interrupting mid-upgrade leaves the old binary intact.
- **No shell exec.** No curl/tar/cp dependencies. All download + extract
  uses Go stdlib.
- **No silent downgrade.** Tags are parsed as semver; dev builds are
  always treated as "older than any release" so `spwn upgrade` never
  overwrites a newer local build with an older release.

### Background version check

Every CLI invocation spawns a background goroutine that calls
`base.CheckLatestVersion(24h)`. It writes to `~/.spwn/.version-check`
(timestamp + latest tag, 24h TTL). When a newer version exists, a single
yellow hint is printed after the command output:

```
  ↑ spwn v1.2.3 available (you have v1.1.0) - run `spwn upgrade`
```

Disable with `SPWN_NO_UPDATE_CHECK=1`.

### `spwn upgrade` CLI surface

```
spwn upgrade                 # install latest stable
spwn upgrade --check         # report without installing
spwn upgrade --channel beta  # include prereleases
spwn upgrade --force         # reinstall current version
```

---

## Tauri desktop auto-update

The desktop app uses the official
[`tauri-plugin-updater`](https://v2.tauri.app/plugin/updater/) plugin,
which:

1. On app launch (wired in `components/app-shell.tsx` → `checkForUpdatesOnStartup()`)
2. Fetches the manifest URL from `tauri.conf.json`:
   `https://github.com/jterrazz/spwn/releases/latest/download/latest.json`
3. Compares against the running app's bundled version.
4. If newer, verifies the **Ed25519 signature** against the public key
   embedded in the app.
5. Shows a native confirmation dialog; on approval, downloads, installs,
   and relaunches.

### Signing keys (one-time setup)

Generate the keypair once:

```bash
cd apps/web
pnpm tauri signer generate -w ~/.tauri/spwn-web.key
```

This produces two files:
- **Private key** (`~/.tauri/spwn-web.key`) - stored in
  GitHub Secrets as `TAURI_SIGNING_PRIVATE_KEY`.
- **Public key** (printed to stdout) - pasted into `tauri.conf.json`
  under `plugins.updater.pubkey`.

Also set `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` if you passphrase-protected
the key.

### `latest.json` generation

`tauri-apps/tauri-action@v0` generates and uploads the manifest when
`includeUpdaterJson: true` is set in the workflow. It looks like:

```json
{
  "version": "0.11.0",
  "notes": "...",
  "pub_date": "2026-04-05T...",
  "platforms": {
    "darwin-aarch64": {
      "signature": "dW50cnVzdGVkIGNvbW1lbnQ6IHNpZ25hdHVyZSBmcm9tIHRhdXJp...",
      "url": "https://github.com/.../spwn-web_0.11.0_aarch64.app.tar.gz"
    },
    "darwin-x86_64":  { "signature": "...", "url": "..." },
    "linux-x86_64":   { "signature": "...", "url": "..." }
  }
}
```

The `latest.json` is attached to the release and served via GitHub's
download redirector, so `releases/latest/download/latest.json` always
points at the most recent published manifest.

---

## Channels

| Channel   | Git tag pattern         | Who picks it up                   |
| --------- | ----------------------- | --------------------------------- |
| `stable`  | `vX.Y.Z`                | Default. GitHub's "latest" filter |
| `beta`    | `vX.Y.Z-beta.N`         | `spwn upgrade --channel beta`     |
| `nightly` | `vX.Y.Z-nightly.STAMP`  | Reserved; not yet wired           |

GoReleaser's `prerelease: auto` automatically marks any tag with a hyphen
as a GitHub prerelease, so the stable "latest" endpoint still returns only
`vX.Y.Z` tags.

---

## Testing

### Unit tests

```
packages/base/update/
  version_test.go         - semver parse + comparison
  release_test.go         - GitHub API client, channel resolution, asset lookup
  download_test.go        - download retries, checksums parse, SHA256 verify
  install_test.go         - tar.gz extract (incl. nested paths), atomic replace
  update_test.go          - end-to-end: fake release server + checksum
                            enforcement + mismatch rejection + dev-build handling
```

Run all with `go test ./packages/base/update/...`.

### Release dry-run

Check what the next release would contain without publishing:

```bash
goreleaser release --snapshot --clean
```

This builds everything locally under `./dist/`.

### Updater dry-run (Tauri)

After a signed build is on GitHub, simulate what the app would do:

```bash
curl -s https://github.com/jterrazz/spwn/releases/latest/download/latest.json | jq
```

Verify the version, platform URLs, and that signatures are populated.

---

## Security notes

- **Never commit `~/.tauri/spwn-web.key`.** It's the only thing
  between the public and arbitrary code execution inside every user's
  web UI instance. GitHub Secrets is the correct home.
- **Pub-key rotation.** If the private key is ever leaked, publish a new
  release with a rotated pubkey AND tell users to reinstall manually (the
  old app cannot verify signatures from the new key).
- **CLI checksums are not Ed25519-signed** - they protect against
  transport errors and accidental tampering. If you need stronger
  guarantees, add cosign/minisign signatures and verify in `update.Apply`.

---

## File map

```
.github/workflows/release.yaml          # CI pipeline
.goreleaser.yml                         # CLI cross-build + checksums
apps/cli/upgrade.go                     # spwn upgrade command
apps/cli/version_check.go               # background version check
packages/base/update/                 # reusable update logic + tests
apps/web/src-tauri/tauri.conf.json        # updater endpoint + pubkey
apps/web/src-tauri/Cargo.toml             # tauri-plugin-updater dep
apps/web/src-tauri/src/lib.rs             # plugin registration
apps/web/src-tauri/capabilities/default.json  # updater:default
apps/web/src/lib/tauri-updater.ts         # frontend check + dialog
apps/web/src/components/app-shell.tsx     # startup hook
docs/contributing/update-system.md      # this file
docs/contributing/releasing.md          # release runbook
```
