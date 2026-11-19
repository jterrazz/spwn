# Release

Spwn is not deployed: it is released. This chapter is everything needed to cut
one: the keys a clone signs with, the tag that starts the pipeline, how to
verify what came out, and how to undo it. The machinery underneath (the
GitHub-only distribution channel, the updater manifest, the checksum
verification) is [Update system](../concepts/update-system.md).

There is nothing to run and nothing to keep alive: the artifacts live in a
GitHub Release, and a user's `spwn upgrade` or the Tauri updater does the rest.

## One-time setup (do this once per clone)

### 1. The Tauri updater key

These sign the updater manifest of the desktop app in `apps/web`, which no job builds today. They live beside the signing secrets, in Infisical (§2), and a job that signs the bundle reads them alone, naming them to `.github/actions/signing-secrets`.

| Secret                               | Purpose                                 |
| ------------------------------------ | --------------------------------------- |
| `TAURI_SIGNING_PRIVATE_KEY`          | Ed25519 private key for .app signatures |
| `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` | Passphrase (if you set one)             |

Generate the keypair:

```bash
cd apps/web
pnpm tauri signer generate -w ~/.tauri/spwn-web.key
```

Output:

```text
Your keypair was generated successfully
Private: ~/.tauri/spwn-web.key      (KEEP SECRET)
Public:  dW50cnVzdGVkIGNvbW1lbnQ6IG1pbmlzaWduIHB1YmxpYyBrZXk6...
```

Paste the **private** key contents into the Infisical secret `TAURI_SIGNING_PRIVATE_KEY`, and its passphrase into `TAURI_SIGNING_PRIVATE_KEY_PASSWORD`, in the folder §2 names.

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

### 2. Signing and notarization

Every macOS artefact the Publish workflow ships from `main` or a `v*` tag is signed with a Developer ID Application certificate, with the hardened runtime and a secure timestamp, and notarized by Apple with `notarytool`, by Apple ID: spwn's two darwin binaries, signed and notarized by GoReleaser's post-build hook (`scripts/sign-macos.sh binary`) before it archives them; station's two darwin binaries, notarized inside a zip that is not shipped, since a bare binary cannot be stapled; and the two apps, `Station.app` and the window's `Spwn.app`, each notarized, then stapled before it is zipped into `Station.app.zip` and `Spwn_darwin_arm64.zip`.

The workflow stores no secret in GitHub. Each job that signs reads the secrets it names from Infisical, through the job's GitHub OIDC token, with `.github/actions/signing-secrets`: the machine identity `studio-ci`, project `studio-ci`, environment `prod`, folder `/spwn`. The project and the folder are the workflow's `SIGNING_SECRETS_PROJECT` and `SIGNING_SECRETS_PATH`. Every value is masked in the log, and a secret a job does not name is never in its environment.

| Secret                       | What it is                                                                                   | How to make it                                                                                                                                                                                                            |
| ---------------------------- | -------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `APPLE_CERTIFICATE`          | The Developer ID Application certificate and its private key, a `.p12` in base64, one line   | Create the certificate in the Apple Developer account (Certificates, type Developer ID Application) from a signing request made by Keychain Access, then export it with its key as a `.p12`; `base64 -i developer-id.p12` |
| `APPLE_CERTIFICATE_PASSWORD` | The password the `.p12` was exported with                                                    | Chosen at the export                                                                                                                                                                                                      |
| `APPLE_SIGNING_IDENTITY`     | The identity codesign signs with: its full name, `Developer ID Application: <name> (<team>)` | `security find-identity -v -p codesigning` on the Mac that holds the certificate; its SHA-1 works too                                                                                                                     |
| `APPLE_ID`                   | The Apple ID notarytool submits as, a member of the team                                     | The account's email                                                                                                                                                                                                       |
| `APPLE_PASSWORD`             | An app-specific password of that Apple ID, never the account's own                           | account.apple.com, Sign-In and Security, App-Specific Passwords                                                                                                                                                           |
| `APPLE_TEAM_ID`              | The ten-character team the certificate belongs to                                            | The Membership details of the Apple Developer account; the identity's name and every signature are checked to carry it                                                                                                    |

The workflow's first job, Signing, runs `scripts/sign-macos.sh require`. Infisical gives a token to `main` and to a `v*` tag alone, so on those two refs a missing secret fails it, naming the missing ones, and nothing is built or released: a tag never ships unsigned. A dry run on any other branch reads nothing and builds unsigned. The Check workflow never signs. The certificate is imported into a keychain the macOS job creates and deletes, with a random password masked in the log, and the decoded certificate lives in the runner's temporary folder only while the import runs.

No artefact carries entitlements: neither `station`, `Station.app` nor `Spwn.app` uses an API the hardened runtime withholds without one. The window starts its programs as plain child processes (`open`, the installer's shell, `station`), reaches Terminal through `open -a Terminal` on a `.command` script and never through Apple Events, and its web view is the system's WKWebView, whose JIT runs in WebKit's own process. The script signs ad hoc with `APPLE_SIGNING_IDENTITY=-` and skips notarization and stapling with `MACOS_NOTARIZE=skip`, to run its logic on a Mac with no certificate:

```sh
APPLE_SIGNING_IDENTITY=- MACOS_NOTARIZE=skip scripts/sign-macos.sh sign machines/.artifacts/menubar/Station.app
```

#### Owner step: open Infisical to spwn's runs

The workflow reads nothing until two things are true, both the owner's:

1. **The secrets are in the folder the workflow reads.** Copy the eight keys of `studio-infrastructure` `/spwn` into project `studio-ci`, environment `prod`, folder `/spwn`; `studio-ci` is the only project CI's identity may view.
2. **spwn's runs present the subject that identity trusts.** The identity `studio-ci` trusts `repo:jterrazz@18480834/*:ref:refs/{heads/main,tags/v*}` with audience `https://github.com/jterrazz`, the immutable form every other `jterrazz` repository uses. spwn still presents the default `repo:jterrazz/spwn:ref:…`; turn on the immutable subject for the repository (GitHub, the repository's OIDC subject claim customization, `use_immutable_subject`), after which it reads `repo:jterrazz@18480834/spwn@1169179413:ref:refs/…`.

The first real proof is then a dry run on `main`, `gh workflow run publish.yaml --ref main`: it reads the secrets, signs and notarizes every artefact, and publishes nothing.

### 3. GoReleaser is already configured

`.goreleaser.yml` is committed; the workflow passes it the default `GITHUB_TOKEN`, and on a signed run the identity the job imported into its keychain and the notarization secrets above, which the spwn build's post hook reads.

## Cutting a release

### 1. Decide the version

Follow semver: `vMAJOR.MINOR.PATCH`. One tag versions both products, and the
line continues after Spwn's last release, `v0.20.0`: the first unified release
is `v0.21.0-alpha.1`. Station's own last release was `v3.1.0` (tagged in its
former repository, `jterrazz/studio-station`); station accepts any version from
`v0.1.0`, so `v0.21.0-alpha.1` passes.

A host on `v0.20.0` is offered it: `spwn upgrade` compares the minor first
(`packages/upgrade/version.go:84`, 20 < 21), so `0.20.0 < 0.21.0-alpha.1` and
`UpdateAvail` is true (`packages/upgrade/update.go:62`). The release is not
marked a pre-release, so the stable channel's `releases/latest` returns it.

- **Patch** (`v0.11.0 → v0.11.1`): bug fixes, no API changes.
- **Minor** (`v0.11.0 → v0.12.0`): new features, backward compatible.
- **Major** (`v0.x.x → v1.0.0`): breaking changes.
- **Prerelease**: suffix with `-beta.N` or `-rc.N`. Marked automatically
  by the tag only: GoReleaser publishes it as a normal release marked latest
  (`prerelease: false`, `make_latest: true`), so `releases/latest` returns it.

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

That's it. The `Publish` workflow runs automatically, and one tag releases
both products of the repository, Spwn and the engine (`machines/`):

```text
.github/workflows/publish.yaml
├─ signing          (ubuntu, arm)   on main or a tag, fails unless every signing secret is read from Infisical
├─ machines-linux   (ubuntu, arm)   station-linux-arm64, station-linux-amd64, cgo off
├─ machines-darwin  (macos-26)      station-darwin-arm64, station-darwin-amd64 (cgo, IOKit), Station.app.zip,
│                                   Spwn_darwin_arm64.zip (the window), each signed and notarized, the apps stapled
└─ publish (GoReleaser, macos-26)   opens the release once both are built: spwn's archives, its darwin
                                    binaries signed and notarized, station's files as extra_files,
                                    one checksums.txt for all
```

The release is opened once, by GoReleaser, and only after both station jobs
succeeded, so a failed station leaves no release behind. Station's binaries
carry its version through `-X .../apps/engine/wire.version=<tag>`, as its own
release did. The window's `Spwn.app` is built by `make window` and versioned the same way, from `git describe`; nothing installs it yet, and it has no updater. The desktop app of `apps/web` and its `latest.json` have no job: the shared
`release-tauri.yaml` they called is deleted, and a job that builds them
again signs them as the macOS job does.

A manual run of the workflow (`workflow_dispatch`) is a dry run: every job
builds, signed and notarized on `main`, unsigned on any other branch,
GoReleaser runs with `--snapshot`, and nothing is published.

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

## Dry runs

### CLI build (no upload)

```bash
make -C machines build-linux
mkdir -p .artifacts/station
cp machines/.artifacts/go/station-linux-arm64 .artifacts/station/
goreleaser release --snapshot --clean
# Artifacts land in ./.artifacts/goreleaser/
```

GoReleaser reads station's files from `.artifacts/station/` as extra files,
and a glob that matches nothing is an error, so a local snapshot needs at
least one station binary there.

### Tauri build (no upload)

```bash
cd apps/web
pnpm tauri build
# Bundles land in .artifacts/cargo/release/bundle/
```

### Updater manifest

```bash
curl -s https://github.com/jterrazz/spwn/releases/latest/download/latest.json | jq
```

Expect:

- `version` matches the latest release
- `platforms.darwin-aarch64.signature` is non-empty
- `platforms.*.url` points at valid assets

## Troubleshooting

### "signature mismatch" in Tauri updater logs

Either:

1. The app was built with a different private key than the one producing
   signatures. Re-check `TAURI_SIGNING_PRIVATE_KEY` in Infisical.
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

## Key rotation

If the Tauri private key leaks:

1. Generate a new keypair.
2. Update `TAURI_SIGNING_PRIVATE_KEY` in Infisical.
3. Update `pubkey` in `tauri.conf.json`.
4. Cut a new release - users running the old app **cannot** auto-update
   to it (signatures won't verify with the old pubkey).
5. Publish a manual-install notice on the website. Users must download
   the new bundle directly and replace their app.

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

## Related

- [Update system](../concepts/update-system.md): the pipeline, the artifacts, and how a
  running install discovers a new version.
- [Develop](develop.md): the loop and the gates a change passes before
  it is ever in a tag.
