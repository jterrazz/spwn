# Release Runbook

Everything needed to cut a new spwn release. The full pipeline runs from a
single `git push --tags`.

For architecture/security details see [`update-system.md`](./update-system.md).

---

## One-time setup (do this once per clone)

### 1. GitHub repository secrets

| Secret                                  | Purpose                                |
| --------------------------------------- | -------------------------------------- |
| `TAURI_SIGNING_PRIVATE_KEY`             | Ed25519 private key for .app signatures |
| `TAURI_SIGNING_PRIVATE_KEY_PASSWORD`    | Passphrase (if you set one)            |

Generate the keypair:

```bash
cd apps/web
pnpm tauri signer generate -w ~/.tauri/spwn-web.key
```

Output:
```
Your keypair was generated successfully
Private: ~/.tauri/spwn-web.key      (KEEP SECRET)
Public:  dW50cnVzdGVkIGNvbW1lbnQ6IG1pbmlzaWduIHB1YmxpYyBrZXk6...
```

Paste the **private** key contents into `TAURI_SIGNING_PRIVATE_KEY`:

```bash
gh secret set TAURI_SIGNING_PRIVATE_KEY < ~/.tauri/spwn-web.key
gh secret set TAURI_SIGNING_PRIVATE_KEY_PASSWORD  # prompts
```

Paste the **public** key into `apps/web/src-tauri/tauri.conf.json`:

```json
"plugins": {
  "updater": {
    "pubkey": "dW50cnVzdGVkIGNvbW1lbnQ6IG1pbmlzaWduIHB1YmxpYyBrZXk6...",
    ...
  }
}
```

Commit that change **once**. Never rotate the pubkey casually - see
[Key rotation](#key-rotation) below.

### 2. GoReleaser is already configured

No extra setup needed. `.goreleaser.yml` is committed and the workflow
uses the default `GITHUB_TOKEN`.

---

## Cutting a release

### 1. Decide the version

Follow semver: `vMAJOR.MINOR.PATCH`.

- **Patch** (`v0.11.0 → v0.11.1`): bug fixes, no API changes.
- **Minor** (`v0.11.0 → v0.12.0`): new features, backward compatible.
- **Major** (`v0.x.x → v1.0.0`): breaking changes.
- **Prerelease**: suffix with `-beta.N` or `-rc.N`. Marked automatically
  by GoReleaser (`prerelease: auto`), not picked up by `--channel stable`.

### 2. Bump version files (optional but recommended)

Keep these in sync with the tag you're about to push:

```bash
# desktop Tauri app
vim apps/web/src-tauri/tauri.conf.json   # "version"
vim apps/web/src-tauri/Cargo.toml        # [package] version
```

The CLI version is injected at build time via ldflags, so no source file
bump is needed for the Go binary.

### 3. Tag and push

```bash
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin main
git push origin v1.2.3
```

That's it. The `Release` workflow runs automatically and takes ~8 min:

```
.github/workflows/release.yaml
├─ release (GoReleaser)           builds CLI archives + checksums.txt
└─ tauri-app (matrix)             builds .app / .dmg / .AppImage
                                  signs bundles, attaches latest.json
```

### 4. Verify

Monitor the workflow:

```bash
gh run watch
```

After it finishes:

```bash
# CLI assets present?
gh release view v1.2.3 --json assets -q '.assets[].name'

# Checksums file present?
gh release view v1.2.3 --json assets -q '.assets[] | select(.name=="checksums.txt")'

# Updater manifest for Tauri present?
curl -sI https://github.com/jterrazz/spwn/releases/latest/download/latest.json
```

Then actually test the upgrade path on a dev machine:

```bash
spwn upgrade --check           # should show the new version
spwn upgrade                   # should download, verify, swap
spwn --version                 # should print v1.2.3
```

### 5. Announce

Nothing to do here - users running the CLI or web UI will be notified
automatically via the background check / Tauri updater.

---

## Dry runs

### CLI build (no upload)

```bash
goreleaser release --snapshot --clean
# Artifacts land in ./dist/
```

### Tauri build (no upload)

```bash
cd apps/web
pnpm tauri build
# Bundles land in src-tauri/target/release/bundle/
```

### Updater manifest

```bash
curl -s https://github.com/jterrazz/spwn/releases/latest/download/latest.json | jq
```

Expect:
- `version` matches the latest release
- `platforms.darwin-aarch64.signature` is non-empty
- `platforms.*.url` points at valid assets

---

## Troubleshooting

### "signature mismatch" in Tauri updater logs

Either:
1. The app was built with a different private key than the one producing
   signatures. Re-check `TAURI_SIGNING_PRIVATE_KEY` in GitHub Secrets.
2. The embedded pubkey in `tauri.conf.json` doesn't match the signing
   keypair. Regenerate and sync.

### `spwn upgrade` fails with "no checksum entry"

The release is missing `checksums.txt` or the archive names don't match
what's in the checksums file. GoReleaser generates this automatically;
if it's absent, the workflow likely failed. Re-run from the GitHub UI.

### `spwn upgrade` says "cannot write to /usr/local/bin/spwn"

The binary is installed in a location the current user can't write to.
Either:
- Reinstall to `~/.local/bin/spwn` via `make install` (default path)
- Or re-run with elevated privileges

### Tag was pushed but workflow didn't trigger

Check the tag pattern. The workflow fires on `v*` so `1.2.3` without the
`v` prefix does not trigger. Delete and re-tag:

```bash
git tag -d v1.2.3
git push origin :refs/tags/v1.2.3
git tag -a v1.2.3 -m "..."
git push origin v1.2.3
```

---

## Key rotation

If the Tauri private key leaks:

1. Generate a new keypair.
2. Update `TAURI_SIGNING_PRIVATE_KEY` in GitHub Secrets.
3. Update `pubkey` in `tauri.conf.json`.
4. Cut a new release - users running the old app **cannot** auto-update
   to it (signatures won't verify with the old pubkey).
5. Publish a manual-install notice on the website. Users must download
   the new bundle directly and replace their app.

---

## Reverting a release

Delete the tag AND the release, then re-push corrected artifacts:

```bash
# Delete tag locally and remote
git tag -d v1.2.3
git push origin :refs/tags/v1.2.3

# Delete the GitHub release
gh release delete v1.2.3 --yes
```

The background check caches for 24h, so existing installs will continue
to see the deleted version for up to a day.
