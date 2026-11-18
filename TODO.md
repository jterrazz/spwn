# TODO

Decided with the owner and left for later, each with what starts it.

## Blueprints: each machine's install, declared (the next major project)
The owner's goal (2026-11-10): a folder of configuration files says what
every machine of the fleet installs, the MacBook, the Mac mini and each kind of
VM, the way an Ansible playbook declares a host, and the app edits the same
files visually. "blueprint" is the working word: `profile` and `recipe` are
taken (glossary). It starts on the new MacBook, which stays minimal, with the
work in workspaces.

- **What a blueprint holds**: catalogue ids only (apps, settings with their
  non-secret values, dotfiles, the image a workspace starts from). A secret is
  named by its vault reference, never written. The catalogue stays the module
  library; a blueprint adds no install logic.
- **Layers**: `base`, then one per role (the owner's Mac, an always-on Mac, a
  Linux workspace, a macOS workspace, a server), then one machine's overrides.
  The last layer wins, and the app shows where each line comes from.
- **Where they live**: open. Not a git repository of their own (the owner
  declined, 2026-11-10); the engine reads them from one path.
- **Plan, then apply**: `station plan` prints the difference between the
  blueprint and the machine (to install, to set, extra), and `station apply`
  does it, idempotent and check-first like `init`. Removing what a blueprint
  does not list is a separate flag, never the default.
- **Drift**: the engine compares on its own and the menu bar shows "3 apps
  missing, 1 extra" on the machine's card, with Apply.
- **Images**: a workspace role's blueprint is baked into its image by the
  recipe, so a new VM is ready at boot instead of installing on first start.
- **The minimal host**: the owner's Mac keeps station, tart, Tailscale, Claude
  Code, git, gh, gpg, Ghostty and Herdr, and lets Claude iterate on station
  from the host. Everything else installs in workspaces.
- **The visual side**: a Blueprints page per role, with the catalogue as
  toggles and values in forms. It writes the files, which stay the source of
  truth, and it is designed with the owner (prototype, then stories) before any
  lot.
- **Claude Code's settings are a blueprint setting** (the owner, 2026-11-10).
  Today station never touches `~/.claude/settings.json`, so the new MacBook's
  were set by hand: session
  retention (`cleanupPeriodDays: 365`, against a default that erased the old
  Mac's sessions past 30 days), the auto-compact window, notifications, the
  theme, the statusline and the `no-local-poll` hook (rtk's hook only where
  rtk is installed). A blueprint layer declares them and `apply` writes them,
  the owner's keys beside station's. Open: the Claude theme is copied today by
  studio-os's `apps/herdr` install, so one of the two owns it; and whether
  moving a checkout's sessions and memory to its new path (done by hand on
  2026-11-10) is a station gesture.
- **git's identity**: no setting writes `user.name` and `user.email`; Accounts
  only reports them unset, and `git-signing` reads them from the preferences
  without writing them. The new MacBook's were set by hand (2026-11-10); a
  blueprint names them.

It replaces the idea of `station apps install --all`. Starts once the new
MacBook runs station v3 and the owner opens the design.

## Install and upgrade for customers (designed 2026-11-16)
Today `scripts/install.sh` is the only install and its re-run the only self-upgrade ([ADR-041](docs/decisions/041-the-lock-is-what-apply-installs.md)): no version folder, no rollback, a checksum from the same origin as the binary (integrity, not authenticity). The design below makes one product version stamp every piece (CLI and engine, guest binary, Station.app, the window, the pinned dependencies), released together. Each lot is one maker, in order.

- **Two doors, one layout**: `curl -fsSL <domain>/install.sh | sh` (the primary door) and a Homebrew cask in `jterrazz/homebrew-station` (`auto_updates true`, so `brew upgrade` leaves it). Both land `~/.local/share/station/versions/<v>/` (`bin/station`, `guest/station-linux-<arch>`, both apps), `~/.local/bin/station` as a symlink flipped atomically, and a receipt `install.json` (`channel: script|brew|dev`). The script unpacks into `<v>.partial` and renames; a `make provision` build is `dev` and never updates. On Linux the script installs the binary only, and inside a VM its Mac's engine owns it.
- **A signed manifest**: each release ships `manifest.json` signed by an ed25519 key whose public half is embedded, and a stable `latest.json`. The script verifies the signature, then each asset's sha256. An older binary refuses an unknown major and says to reinstall. Customers also need a Developer ID, the hardened runtime and notarisation; the cask needs them to pass Gatekeeper.
- **The update check**: the engine fetches `latest.json` every 12 h (at start and on wake, up to 1 h of jitter) into `~/.cache/station/update.json`. A command reads that cache with no network, and prints at most one stderr line a day per version, on a TTY only. Silent for `--json`, a pipe, `CI`, a VM guest and the `dev` channel. Offline keeps the last cache; after 14 days Needs you says `updates not checked since <date>`. The menu bar and the window show one upkeep row the badge never counts. Opt-outs: `preferences.json` `updates: {check, apply: night|ask|never}`, `STATION_NO_UPDATE_CHECK=1` (check and line), `STATION_DISABLE_UPDATES=1` (every path, `upgrade --self` included).
- **The engine is the one updater**; no Sparkle and no Tauri updater (a second channel that cannot move the engine or the guests, and lockstep keeps `page.Revision` compatible). It runs station's own step first in the 03:00 night ([ADR-042](docs/decisions/042-versions-are-semver-ranges-applied-nightly.md)), or on `station upgrade --self`, in this order, each with its health gate and rollback:
  1. Engine, VMs running: verify the manifest and assets, snapshot the stores, take the machine's turn, write `runs/<vm>.json v1`, flip the symlink, `launchctl kickstart -k`. The VMs survive and `Adopt` re-attaches them (`specs/cli/engine/adopts.spec.yaml`). Gate in 60 s: hello answers the new version, every run adopted with its pid, each guest reconnected. A failed gate or three exits in 5 min flips back, restores the snapshot only if a store's version moved, records `update.rolled-back <v>` and skips that version.
  2. Station.app: the catalog script's rename lifted into the engine, then `station.updated`; the app relaunches unless its panel is open. Rollback restores `.old`.
  3. The window: `bundle.active` on and notarised, swapped the same way, "Restart to update" while open.
  4. The guest binary, over the engine's ssh into `/usr/local/bin/station.new`, sha256 checked in the guest, `station.prev` kept, hello gate in 20 s. A stopped VM is pushed at its next boot, before the boot phase. The guest's own `upgrade --self` refuses.
  5. Images: `rebuild waiting` is said, never done unasked (`images.rebuild: ask`, `night` opt-in); two versions are kept.
  6. Dependencies: pinned in the manifest with a sha256, into `deps/<name>/<version>/` behind a `current` symlink, moved only with a station release. A running VM keeps its tart (its wrapper names the versioned path); prune only what no `runs/*.json` names. The unchecked installers (Homebrew, mise.run, herdr) become a `script` method with `url` and `sha256`.
- **Version skew**: CLI and engine are one binary (skew only between flip and restart; after 60 s the CLI says to run `station engine start`). A guest may lag at any version at or above `api.Floor`, the engine pushes it up, and Floor rises only after a release that pushes guests. The menu bar and the window are lockstep, the engine accepting one `page` revision back. The engine reads `runs/<vm>.json` of N-1 and N+1, and a file it cannot read is `foreign`: never killed, never adopted, shown. Stores expand then contract, a breaking migration spanning two releases, since a store refuses a newer file today.
- **tart stays an external dependency the user installs** (the owner, 2026-11-16: "pour tart on garde en dépendance, on dira qu'il faut l'installer (cf le machinefile du mac initial, ça sera une dépendance obligatoire)"). It is a required app of the Mac's initial Machinefile, declared there and never bundled in the release. What still applies, in U8: its version is pinned and its sha256 checked in the catalog (today `catalog/packages/containers-vms/tart.yaml` takes `latest` with no checksum). Open: whether the `~/.local/bin/tart` launcher (`TART_HOME`) stays on the PATH. tart is `openai/tart` under FSL-1.1-ALv2 since 2026-04-07, whose Competing Use clause may block customers. A second `Hypervisor` adapter (`vz`, MIT, behind the port; Lima, Apple `container`) waits for that: the port gains `Capabilities()` and each VM record names its `hypervisor`, a VM always booting by its maker. Starts when the licence or the owner of tart turns bad, or a customer is blocked by it.
- **The 26 scenarios**, each a spec of its tier (the fake world proves sequences, stubs argv, VMs the real restart):
  - usecases: update-check/cache-not-network, says-once-a-day, silent-cases, offline-keeps-cache; self-upgrade/refuses-bad-asset, gate-fails-rolls-back, crash-loop-rolls-back, store-migration-rollback, night-runs-station-first, apply-never; adopt/foreign-run-file; guest/push-on-upgrade, push-fails-keeps-prev, stopped-vm-updated-at-boot; menubar/relaunch-after-swap; deps/tart-move-spares-running-vm; hypervisor/capability-refusal, vm-keeps-its-maker.
  - adapters: install/script-verifies-signature, cask-same-layout; adopt/pid-reused-is-not-adopted; deps/tart-pinned-sha.
  - cli: install/script-versioned-layout; self-upgrade/engine-restart-keeps-vms (extends `adopts.spec.yaml`).
  - machine (tiers 2 and 3, on a local release server signed with a test key, `STATION_UPDATE_URL` and `STATION_UPDATE_PUBKEY` honoured by test builds only): e2e/upgrade-real-vm, e2e/fresh-mac-install-and-update.
- **Lots**:
  - U1: `Adopt` compares `p_starttime` against `runs/<vm>.json v1` (the `.pid` as fallback).
  - U2: the signed `manifest.json` and `latest.json` in publish.yaml.
  - U3: the versioned layout and receipt in `install.sh`; the plist names the symlink; prune keeps 2.
  - U4: the update check: fetch, cache, line, silences, preferences.
  - U5: `station upgrade --self` and the night's first step, with download, verify, snapshot, flip, kickstart, gate, rollback. It carries the self-replace risk: the release asset with its checksum, replaced by a rename, then the engine restarted (`apps/engine/images/calls/binary.go`'s download lifted to a shared adapter). `station upgrade` today updates what the Machinefile carries, not station.
  - U6: the guest push, at upgrade and at boot.
  - U7: the menu bar's lockstep swap in the engine, the `station.updated` event, the relaunch.
  - U8: dependencies pinned in the manifest (tart's version and sha256 first); the installers with sha256; the launcher question above.
  - U9: `Hypervisor.Capabilities()` and the per-VM `hypervisor` field.
  - U10: Developer ID signing and notarisation of the CLI zip, Station.app and the window.
  - U11: the Homebrew cask, bumped by publish.yaml.
  - U12: tier 2 and 3 upgrade runs.
  - U13 (later): a `vz` adapter spike behind the port.
- **Open owner questions** (none answered yet):
  1. One updater for the cask too, or brew owns brew installs? Recommendation: station updates both, since brew cannot run the engine restart or the guest push.
  3. The default apply mode. Recommendation: `night`, with `ask` for teams that change-manage.
  4. A beta channel? Recommendation: not before the first customer; `stable` only, plus a `--version` pin.
  5. The Apple Developer ID entity (personal or the company that sells station). Recommendation: the selling entity, decided before U10.
  6. Images rebuilt at night? Recommendation: `ask`.
  7. Start L1 and L3 of the Go audit now. Recommendation: yes, their findings shape the other lots.

Starts with U1, the owner's go, and the answers above for U2 onward.

## Go code: lint and modernisation (audited 2026-11-16)
An audit of `apps/engine` at `3e950d809` found the obvious libraries already in use, so the gains are two defects, stdlib modernisation and lint gaps, not new libraries (the architecture rules allow a third-party import only in adapters, `wire`, the presentation packages and `platform`).

- **Two defects**: `settings/diff.go` `script()` builds an `(n+1)x(m+1)` LCS table, about 72 MB for a 3,000-line `~/.zshrc`, on every `settings diff` and `plan`. `ui/atom/text.go` (`Middle`) and `value.go` (`path`) keep `width-1` runes but measure in cells, so a CJK or emoji name overflows its width.
- **Lint gaps**: `make lint` runs `GOOS=linux` only, so 13 darwin files (1,002 lines, 3 cgo) are never linted; `modernize` is off (247 `errors.As`, 56 `range strings.Split`, 28 `sort.*`); no `govulncheck` runs anywhere. To add: `nolintlint`, `nilnesserr`, `errchkjson`, `bodyclose`, `noctx`, `fatcontext`, `usetesting`, `durationcheck`, `makezero`, `wastedassign`, `unparam`, each with its `value:` or `hazard:` comment. Not `depguard` nor `forbidigo`: `specs/architecture` owns imports and symbols.
- **Hand-rolled on purpose**: semver ranges, YAML decode, schemas, atomic writes, process supervision, retries on an injected clock, TOML surgery, ssh, archives, XDG paths, the JSON-RPC frames and the tart adapter stay, because each does what its library does not (vendor versions, the owner's files kept as written, the fenced env, the wire contract) or sits in a layer that refuses third-party imports.
- **Lots, in order, each one maker, each with a cold `make check` and goldens exact**:
  - L1: enable `modernize` and `nolintlint`; apply `--fix` except `errorsastype`.
  - L3: a `GOOS=darwin` pass in `make lint` (cgo on a Mac, `CGO_ENABLED=0` in CI), and its findings.
  - L4: `settings/diff.go` trims the common prefix and suffix before the table; a 5,000-line benchmark.
  - L5: `ui/atom` truncation on `x/ansi` (`Truncate`, `TruncateLeft`, `Cut`), `ansi` added to the `ui/atom` row of `importTable`, one wide-rune fixture.
  - L6: one flock on `unix.Flock` in `platform/local/fs.go`, `ExclusiveWithin` over `Exclusive`, the lock tests under `synctest`.
  - L2: the `errors.As` to `errors.AsType` sweep (247 sites).
  - L7: a binary plist's program read through `plutil -convert xml1` in `launchd/plist.go`.
  - L8: the `tool` directives into a `tools` member module, `govulncheck` in `make check` and CI.
  - L9: the roster additions above, with their findings fixed (split past about 40).
  - L10 (optional): `.tar.xz` and `.tar.zst` release assets.

Starts with L1 and L3, then L4 to L6, then L2 and L7 to L9, L10 only if the owner wants the feature.

## The next minor release: the v2 words go
The hidden aliases (`status`, `start`, `stop`, `tools`, `config`,
`run`, `probe`, `--always-on`) and `station.status/v1` are kept for one release ([Commands](docs/reference/commands.md#hidden-aliases)). The next
minor removes them, with the `machines supervise` retirement shim and the
retired preference names (`here`, `hosts`, `services`), once
`studio doctor` reads `station.overview.read/v1`.

## The test VMs show in Disk
Tier 3 (`specs/machine/macos/tier.sh`) makes its macOS VM with tart directly,
so the engine never knows it: `station-e2e-macos-<hash>` held 31 GB and showed
nowhere in the app (2026-11-10). Disk lists a `station-e2e-*` or
`station-play` VM as a test VM, with its size and erase; it never becomes a
machine. Starts with the next tier 3 run.

## The goldens stay small
The menu bar's snapshot goldens weighed 2.8 GB across history and pushed the
branch past GitHub's 2 GB push limit; their history was dropped on
2026-11-10. A golden is compressed or drawn at 1x before it is committed, so
one story's update stays in kilobytes. Starts at the next golden change.

## A password manager of station's own
The owner's idea (2026-11-10): rather than adopt Bitwarden or 1Password, station
carries its own password manager, a second adapter of `accounts/contract.Vault`,
so sharing credentials between VMs and Macs is one gesture of the app:
a credential is shared with a VM for a term, scoped
(`operator:*` tokens per VM, beyond `session:*`), and taken back. A blueprint
names a credential by its reference in this vault. Not designed: the
encryption at rest, the unlock, the sync between Macs and the recovery are
decided with the owner before any lot. Starts when a VM first needs a token
that is not a browser session.

## The macOS VM and the iPhone (on the MacBook M5 Pro, from 2026-11-07)
A `macos` image beside `linux`: pull `macos-tahoe-xcode:27`,
`--provisioning-opts` instead of admin/admin, `kern.hv_vmm_present` for the
kind, `station guest` on macOS (`lsof`), a build deployed to the iPhone from a
workspace made `--network bridged` (pairing: cable on the Mac, or Accessory
Access; unproven). No Tailscale inside. Then the recipe and its first boot, and
the apps that live on macOS VMs only (xcode, cocoapods, xcodegen,
android-studio, lens) install there. If the iPhone cannot be reached from the
VM, the owner decides how the install step reaches it (an Xcode on the Mac is
not an option he has accepted).

## Machine test tiers: the nightly
Tiers 2 and 3 are built (`make e2e-machine`, [Test](docs/guides/test.md#the-machine-specs)).
What waits is tier 3 as a nightly on the Mac mini, once a tart VM may run
there.

## The Mac mini as an always-on Mac
Installing station there is the owner's go, never a lot's. When its disk is
upgraded: `station init --name mac-mini` there, the Always-on Mac settings,
the infrastructure moved from OrbStack to a server (`machines create infra
--server --check https://infra:6443/readyz`), images built there. Ask the
owner before any test on it.

## The menu bar's stories left after v4
Fourteen gap ids wait for a design before they are stories (STORIES.md,
"Gaps"). Each starts as a lot of its own once the owner asks for it.

## The page kit's agent item types
The page kit (`apps/engine/page/page.go`) still has the item types an agent
drew: `Message`, `Steps`, `PR` and `Reply`. No page of the engine sends one;
only `apps/engine/page/every_field_test.go` and the Swift decoder
(`apps/menubar/sources/station-kit/engine/Client.swift`) hold them. They go in
a later lot, with the fixture and the golden that carry them.

## Permanent agents
Another binary owns agents, and what each machine holds of them
([ADR-052](docs/decisions/052-station-keeps-no-agents.md)); station designs none of it.

## The Herdr iPhone app over SSH
TEMPORARY, revert when the permanent machine session exists: the Herdr
iPhone app reaches the owner's MacBook over SSH (2026-11-11). Herdr core
listens on a local unix socket only, so remote access runs
`~/.local/bin/herdr-core connect` over SSH, as this Mac already does to its
gate VMs. Opened by hand on the MacBook: Remote Login on (System Settings →
General → Sharing; `systemsetup -setremotelogin` needs Full Disk Access),
`/etc/ssh/sshd_config.d/100-tailscale-only.conf` (`AllowUsers` limited to
Tailscale's 100.64.0.0/10 and fd7a:115c:a1e0::/48, key-only), and the
`jterrazz-iphone-herdr` line in `~/.ssh/authorized_keys`. Until then the
settings lot after L2.5 standardizes it as packages (`ssh-authorized-keys`,
`ssh-tailscale-only`, a Remote Login check) and L8.4 puts the key in the
`jterrazz-macbook-pro` Machinefile. Once a permanent machine session holds
everything and the phone connects to it instead: Remote Login off, the
drop-in removed, the key line removed (the file too if it is the only
line), those packages and the key dropped from the MacBook's Machinefile
unless the permanent machine reuses them, and optionally the Mac's host key
removed from the iPhone app.

## Network isolation
Tart Softnet for VMs that need no LAN. Until then a VM behind its Mac reaches
what its Mac binds on every interface, and the other VMs of its Mac (sshd,
Docker's ports), whatever the Tailscale policy says; the Ports section shows
each as Public when it is.

## What the new MacBook met (2026-11-10)
Four gaps found setting up the owner's new Mac and its first workspace, `os`.

- **`git-signing` on a Linux workspace**: the setting is for Macs and macOS
  VMs only, so a commit made in a Linux workspace is unsigned. Until it is
  designed, the host edits and signs, and the workspace only runs the gates.
- **The released images refuse a pull**: `station images update linux`
  answered `the registry refused a pull token for jterrazz/station-linux:
  403`, so the Mac built its own (`images build linux`, 2 min). Either the
  ghcr package is published public, or `update` says it is private and
  names `build`.
- **The `linux` recipe lacks Chromium's libraries**: Playwright's headless
  shell failed on `libatk-1.0.so.0` until `playwright install-deps chromium`
  ran in the workspace. The recipe installs those packages, so a gate with a
  browser runs on a fresh workspace.
- **The release ships no `Station.app`**: the menu bar was built on the host
  with `make menubar-install`, which needs a full Xcode (`actool`, then
  `xcodebuild -runFirstLaunch`), outside the minimal host. The release
  carries the signed bundle, so the host needs no Xcode.

## A workspace that reads the host's checkout
The owner's idea (2026-11-10): share the host's studio-os checkout with a
workspace, the way a Docker volume would, so a gate runs without a push.
Today nothing of a Mac is shared into a VM ([Machines](docs/reference/machines.md#files-and-editors)),
because a writable mount lets a guest write the host's `.git/hooks` or
`Makefile`, which then run on the host. Until it is designed, the host edits
and signs, pushes the branch to the workspace over ssh (`git push os:...`),
and the workspace runs the gate. Open: a read-only mount (`tart run --dir`
with `ro`) copied into the guest before a gate, since a gate writes into its
tree; and whether it is a blueprint's volume. Starts when the push loop
becomes the bottleneck.

## Smaller follow-ups

- `gh`'s `hosts.yml` row says the login is in the keychain's `gh:github.com`: check on a Mac whether `security` may list that entry (it was denied on one read), and add it to the Keychain rows if it can be read.
- `codex`'s `~/.codex/auth.json` row names the file the login lives in: check on a Mac with a Codex login that this is the file's name (`ls ~/.codex`), and fix the path if it is not.
- `claude-desktop`'s `vm_bundles` is level `data`: check whether the app downloads it again after a delete (delete it on a Mac and open the app); if it does, the level becomes `cache`.
- The machine files the window still waits for: its Edit and Allow, which write the owner's files and wait for the owner's word on that.
- The tier-3 proof of `station upgrade`: on a real Mac and a real VM, the managers' real order and that `brew update` refreshes the index now that every bundle runs with `HOMEBREW_NO_AUTO_UPDATE=1` (tier 1 proves the order over the fake world, `specs/usecases/machinefile/upgrade/homebrew-index-first.spec.yaml`, never the argv a real brew gets). Deferred from L2.4, whose plan names it; starts before the lock's release, or with the owner's explicit acceptance of the deferral (ADR-041).
- The lock beside a template in the shared source (`studio-linux.lock`), and an image built from a lock, wait for `images build <machinefile>` (ADR-041).
- While Serve reads off because `tailscale status` failed, the tailnet may still hold station's entries, and a release or a narrowed right then leaves its entry live (`unshareLocked` in `apps/engine/engine/process/share.go` unshares only while Serve is on, `refreshSelf` in `engine.go` reads a failed status as off) until Serve reads on again and the reconciliation removes it; meanwhile its port of the Mac is given to no other share only if the engine's last reading of the tailnet listed it (`carriedLocked` in `share.go`), never when no reading was made since the start. Found by the L4.1 re-verify (2026-11-11); starts with the next change to how the engine reads Serve's state.
- A VM deleted while its engine is down unshares its entries by their port and path only (`ReleaseVM` in `apps/engine/wire/engine.go`), not leaving one carried from another port of the Mac as the engine's `unshareLocked` does, so an entry the owner put by hand on station's Funnel port goes with it. Found by the L4.1 fix round 4 (2026-11-11); starts with the next change to that release.
- The owed retry reads `serve status` under the engine lock at each tick (`payOwedLocked` in `apps/engine/engine/process/share.go`), so a hanging tailscaled can hold the lock up to 2 x 5 s per tick until `refreshSelf` reads Serve off. Found by the L4.1 final verify (2026-11-11); starts with the next change to how the tick reads the tailnet.
- `go test -race ./apps/engine/engine/process/` fails on `TestARefreshReadsAgainOnlyThePageItNames`: `Stream` in `apps/engine/engine/process/subscribe.go` writes `frames[id]` for the next page while the feed goroutine of the one before reads `frames[id]` from the same map (lines 233 and 237), and `make check` runs no `-race` to see it. Pre-existing on main, found by the L4.1 rebase's gate (2026-11-11); starts with the next change to `Stream`.
- A VM's record is written without the engine's lock by four calls of `apps/engine/machines/calls/` (`agents.go:107`, `edit.go:68`, `vms.go:76` and `vms.go:99`, each `writeRecord`), so one can race the engine's own write as a create did before b440a36a. Found by the L4.1 re-verify (2026-11-11); starts with the next change to those calls.
- A server's traffic on its card: a metrics URL a server declares (Prometheus text, read over the tailnet from its Mac), its requests per minute beside the latency ([ADR-028](docs/decisions/028-a-servers-card-shows-latency-not-traffic.md)). Starts when the owner asks for it.
- An image registry (private ghcr) once two Macs build the same image.
- Memory soak: a Linux VM 8 h under load, the Mac's RSS of the VM process.
- `machines edit <vm> --persistent`: turn an ephemeral workspace persistent (rewrite its record; the engine reads it).
- HTTPS mode: if a Mac's `tailscale serve` refuses arbitrary https ports, the default of the `ports` group's `https` becomes `path`.
- Remove the old Tailscale leftovers by hand when wanted: the `tag:station-vm` tag owner, the VM-join OAuth client, each Mac's userspace node (admin console), `brew uninstall tailscale` (the formula; the app replaces it).
- The poweroff intent over a FIFO instead of a console file, if the macOS guest's console differs.
- A second `Notifier` adapter (ntfy, to the phone), and a `Jobs` adapter over systemd for a Linux machine that runs an engine.
- `ansible` installs on Linux by `mise: pipx:ansible` and names no `requires`, unlike `markitdown` and `browser-use` (`requires: [uv]`), so neither the resolution nor `apps install` brings uv or pipx before it. Starts with the answer, read on a Linux VM: does `mise install pipx:ansible` work with neither uv nor pipx there? If not, `ansible` requires what it leans on, as the other two do.
- The password read is `sudo -k -n true`, so a sudoers rule that lets `true` alone, or a few commands, run without a password reads as asking none, and an `apply --yes` off a terminal then meets a password at the install that runs sudo. Starts with a machine whose sudoers rule names commands: the read becomes `sudo -n -l` of the commands the plan's installs run.
- A manager's packages are what its methods list, so what pip installed into a mise python (`pip install` into the runtime, outside uv) is not counted, and `apply --remove` takes that python back with them. Starts when a Machinefile lists python: the holds read adds the runtime's `pip list --not-required`.
- mise's holds read is `mise ls --global`: a runtime a project's own file installed, absent from the global file, is not counted, and taking mise back deletes it with mise's folder. Starts when the owner keeps project runtimes on a machine a Machinefile installed mise on: count `mise ls --installed` beyond the global file.
- A manager command's PATH leaves out nvm's versions, so on a machine whose only node is nvm's, node reads present another way (it is looked for on the owner's PATH) while npm reads as missing to station, and an npm app brings mise's node rather than install into nvm's. Starts with a machine that keeps nvm: decide whether nvm's default node counts as the global one, or is said as another manager's.
- npm and uv packages install through `brew bundle`, and Homebrew filters the environment of what it runs on macOS (unless `HOMEBREW_NO_ENV_FILTERING`), so the `npm install -g` or `uv tool install` it runs may lose the environment station builds: mise's ceiling and its auto-install pins (`MISE_CEILING_PATHS`, `MISE_AUTO_INSTALL=false`), and with them the global version a shim answers with. Installs only: every read and removal runs without the bundle. Starts with the answer, read on a Mac: does a package that `brew bundle` installs, from a folder whose project pins another node, land in the global node's prefix? If not, those lines install by the manager's own command, as their removal does.
- A uv tool runs on the python it was installed with, so a tool installed on a mise python is broken when `apply --remove` takes that python back, and python's holds read does not count it (uv's list is uv's, and uv is its own manager). Starts when a Machinefile lists python and uv together: python holds the uv tools whose interpreter is in one of its global versions' folders.
- What pnpm (`PNPM_HOME`) and corepack install globally lives outside node's prefix, so node's holds read does not count it, while taking node back breaks what runs on it. Starts when a Machinefile lists pnpm: node's holds read adds `pnpm ls -g`.
- A version asked of an `npm`, `uv` or `bun` method is refused as unpinnable ([ADR-038](docs/decisions/038-an-app-is-carried-by-one-method-the-one-its-plan-shows.md)), since their installs carry none. A small lot: each install passes its version (`npm i -g <pkg>@<v>`, `uv tool install <pkg>==<v>`, `bun add -g <pkg>@<v>`), and the refusal goes for that method.
- A bare `station apps update` says `every app is up to date` while an App Store (`mas`) row reads Update available: the App Store updates its own apps, so a bare update leaves them (`UpdateAll` false), and the report says nothing of them.
- Apply passes `apps.install` the method its step chose, not the install, so a package with two methods of the same manager split by a selector `version` range may install by the other one (ADR-038). Starts with the first such package: apply passes the install's key too.
- `apps update --manager cask` updates Homebrew's formulae too, since a manager key reaches every key its package installs by (ADR-038), while its help says "update every app of a manager" and the key names casks. Found by the L3.1 verify (2026-11-11). Starts with the owner's call: the help says that `brew` and `cask` both mean Homebrew, or a key reaches only its own method.
- No built-in package has two methods on one system yet: adoption by another method and `apps move` are proved over the test package `twin` alone (`apps/engine/wire/method_test.go`). Starts with the first such package (an app by cask or by mas, a runtime by mise or by apt): its stories over the real catalog.
- The apps place an app by kind, so a method whose selector names an OS version (`version: '>=15'`) is placed on every version of that OS, while the resolution judges the range (ADR-038). Starts with the first such selector: the shelf places by the machine's system, as the resolution does.
- For an app with a mise method, `present` stops at the first PATH hit (`apps/engine/apps/calls/inventory.go:383`), and mise's shims come first: a shim for a project's pin hides the owner's other copy (`/opt/homebrew/bin/node`), so the app reads not installed and apply installs a global mise copy over it. Found by the L3.1 re-verify (2026-11-11). Starts with a test: the lookup scans the PATH past the folders mise keeps.
- A move from a copy station cannot uninstall ends on a failed "cannot" step if its branch (`apps/engine/apps/calls/operations.go:638`) goes; no test holds it (mutation M9 of the L3.1 re-verify survives). Starts with one test.
- An apt install of a package the owner held (`apt-mark hold`) is kept back without a failure, so the app reads not installed after a run that said it installed, and apt waits for dpkg's lock up to its own `DPkg::Lock::Timeout` (120 s on gate2's Ubuntu, `apt-config dump`) with nothing said meanwhile. Found by the L3.2 security verify (2026-11-11, its F6). Starts with the first install seen to stall or be kept back: the install reads `apt-mark showhold` first and refuses a held package, and passes its own lock timeout.
- Two installs at once (two terminals, or a terminal and the engine) are not serialised: apt's own lock refuses the second apt run, but two releases each read, then rewrite, their record in turn, and the second may write over what the first recorded. Found by the L3.2 security verify (2026-11-11, its F8). Starts with an apps lock of station's own, the one `disk{}.Lock` a VM record takes, held around every install, update and removal.
- A checksums file that names the asset on two lines with two sums vouches by its first line alone (`listed` in `apps/engine/apps/adapter/githubrelease/githubrelease.go`), where the two disagreeing should refuse it. Found by the L3.2 security verify (2026-11-11, its F10, experiment `two-lines-first-wins`). Starts with one test: every line naming the asset is a voucher.
- Every answer of every station GET is cut at the same 512 MiB (`most` in `apps/engine/platform/local/net.go`), the GitHub API's small JSON as much as a release asset, so the one bound is too loose for the first and the only one an asset has. Found by the L3.2 security re-verify (2026-11-11). Starts with a bound each caller gives (`Net.Get` taking its most, a JSON read a few MiB, an asset 512 MiB), the default the small one.
- The package rule accepts an `:arch` (`foo:arm64`, `CheckPackage` in `apps/engine/catalog/names.go`) that the apt adapter matches nowhere: dpkg-query lists the native arch bare, so an owner's package asked again as `foo:arm64` is recorded station's, reads absent so every apply installs it again, and its removal is refused, since `apt-get -s remove` names it bare. Safe-failing, never removing more. Found by the L3.2 re-verify round 3 (2026-11-11, its should 4). Starts with the first package that needs a foreign arch: the rule drops `:arch` until then, or the adapter strips the native arch wherever it matches a package.
- An apt record outlives its package when a removal stops between apt's run and the record's rewrite, and a package the owner installs by hand after station recorded it stays station's, so a later removal takes it. Found by the L3.2 re-verify round 3 (2026-11-11, its nit 5). Starts with a removal that lets go of its records before apt runs, as an install records before it; the package installed again by hand needs dpkg's own log read.
- The argv of bun, mise, brew, npm and uv carry a catalog name after their verb with no `--` and no check by a rule (`apps/engine/apps/adapter/{bun,mise,brew,npm,uv}/`), where apt's and the skills CLI's do both (ADR-040). Not root, and a package can run `shell:` anyway, so no escalation. Found by the L3.2 re-verify round 3 (2026-11-11, its nit 6). Starts with one rule per manager's names in `apps/engine/catalog/names.go` and `--` in every argv that takes one.
- A method's `path` reaches station's search, never the owner's shell: postgres on a Mac (`postgresql@18`, keg-only) reads present by its keg's bin, while a terminal finds no `psql` (mise put it on PATH before L3.3). Found by the L3.3 verify (2026-11-12, its F3). Starts with the owner's call: the shell block adds every installed app's `path`, or the brew method links a keg-only formula (`brew bundle`'s `link: true`).
- The apt names of the Linux command line apps are checked on Ubuntu 24.04 alone: on 22.04 `eza` and `sd` are absent, so apt fails at the install rather than at the plan, and on Debian the eleven apt packages have no method at all (`os: ubuntu`; `eza` is absent from bookworm too). Found by the L3.3 verify (2026-11-12, its F7). Starts with the first image other than 24.04: `version: '24.04'` on eza and sd, and `family: debian` for the packages both ship.
- On Linux, uv's release lands `uv` alone and yazi's `yazi` alone, so `uvx` and `ya`, in the same archives, are not installed (mise installed both before L3.3). Found by the L3.3 verify (2026-11-12, its F8). Starts with one line each: `provides: [uv, uvx]` and `[yazi, ya]`, and every-app.txt naming them.
- `specs/go.mod` and `specs/go.sum` break `@jterrazz/test`'s names rule `specs-root-holds-specifications-and-shared`, which a project cannot relax, so `.studio/baseline.json` holds its two findings (the structural lot, 2026-11-13). Starts with a release of `@jterrazz/test` whose rule accepts a Go module's files at the specs root; the baseline entry goes with it.
