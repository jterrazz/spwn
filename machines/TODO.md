# TODO

Decided with the owner and left for later, each with what starts it.

## Station becomes Machines, past the display name
The owner's ask (2026-11-20): rename station to machines everywhere, now
that Machines is the app's name. The display name landed in v0.21.0-alpha.2
(the UI, the manual and the site say Machines; the window stays "Spwn", the
owner, 2026-11-20). What goes past it waits for one design talk with the
owner on what it does to the commands, which they hold in another session
(the command vision, 2026-11-21).

- **The command grammar**: today `station machines start linux-2`. A binary
  named `machines` reads `machines machines start linux-2`, so the rename
  forces a choice: drop the `machines` noun level (`machines start linux-2`,
  `machines create qa`, and what bare `machines` shows, the fleet or this
  Mac), or one umbrella command (`spwn machines start linux-2`,
  `spwn agents ...`). Either one rewrites the CLI goldens and `words.json`.
  The proposal sent to the owner (2026-11-21) is the umbrella, a flat
  `spwn <noun> <verb>`, with six questions not answered yet, each with its
  recommendation:
  1. Grammar: flat `spwn <noun> <verb>`; Machines and Agents are sections
     of the app and the help, not command levels (the `gh`, `kubectl` model).
  2. Nouns: plural everywhere (`agents`, `worlds`, `machines`, `apps`).
  3. Credentials: Agents drops `auth`, `gate` and `cookie-sync` for
     `spwn accounts`, with one vault.
  4. Daemons: one engine per Mac, absorbing the credential broker and the
     automation daemon.
  5. Automations: they run on Machines' background tasks (the automation
     keeps what to run, Machines when and where).
  6. A world on a machine: yes, `spwn worlds up --on <machine>`.
- **What the talk decides for one bundle** ("One Spwn on a Mac", below):
  whether Spwn.app holds one CLI binary or two, the names of the
  `~/.local/bin` links, the launchd label prefix and the nested menu bar's
  bundle id. None of them changes a Login Items or Local Network row there.
- **The identities**, in the same breaking version: the state, config, data
  and cache folders (`station/`), the launchd labels
  (`com.jterrazz.station.*`), the menu bar's bundle id, the Keychain
  service, the schema ids (`station.*/v1`), the `STATION_*` variables, the
  hello, the zsh and hosts block markers, the release assets (`Station.app`
  and `Station.app.zip`, the one `Station` the retired-word guard still
  allows, `specs/architecture/banned_test.go:70`) and the Go module path. No
  compat code: an installed host takes one written migration (stop the old
  launchd jobs, move the folders, re-add the Keychain items, re-tag the
  guests), proved on a VM before it is handed over.
- **How it lands**: three to five large commits, each a lot tip with
  `make lint` and then the full gate, the `go.mod` commit first, the
  window's recordings recorded once at the end, never a regeneration per
  commit (the runs audit of 2026-11-20).
- **Starts** after the owner's command talk: without the migration the new
  engine orphans every existing VM, the development VMs included.

## One Spwn on a Mac (designed 2026-11-21, L0 measured)
The owner's ask (2026-11-21): macOS lists the product once, where today Login Items shows "Spwn" and the owner's name, Local Network several "station" rows and one "Station", and the Dock two apps. The engine stays usable without the window, so one engine binary has two install modes, exactly one active per Mac. L0 measured what macOS does and wrote the decision, [ADR-073](docs/decisions/073-spwn-app-carries-the-engine-on-a-mac-with-a-screen.md) (Proposed), which owns every measured fact; this item keeps the lots. Apple's sources are [TN3179](https://developer.apple.com/documentation/technotes/tn3179-understanding-local-network-privacy), [Updating helper executables from earlier versions of macOS](https://developer.apple.com/documentation/servicemanagement/updating-helper-executables-from-earlier-versions-of-macos), [SMAppService](https://developer.apple.com/documentation/servicemanagement/smappservice) and `man launchd.plist`.

- **Why each duplicate exists**: two bare binaries in `~/.local/bin` (spwn-web's `public/install.sh`); two legacy LaunchAgents that run `station` by its absolute path (`apps/engine/wire/engine.go:488,509`, `apps/engine/catalog/launch.go:37-62`), both naming `AssociatedBundleIdentifiers` `[com.jterrazz.spwn, com.jterrazz.station.menubar]` since 6dce2c34d, first released in v0.21.0-alpha.3; and two bundles, `Station.app` and `Spwn.app`. Local Network keeps a bare executable as one row per path and signing identifier: on the Mac mini, `~/.local/bin/station` as `a.out` (every Go ad hoc build) and as `station-darwin-arm64` (the Developer ID release), plus one row per e2e run folder (ADR-073, H1). A job recorded before the key keeps its developer parent once the key is added (proved by 6dce2c34d with `sfltool dumpbtm`): that is the owner-name row.
- **Mode A, a Mac with Spwn.app, arm64 and macOS 26 only**: one bundle, `~/Applications/Spwn.app`, Developer ID, hardened, notarized. `Contents/MacOS/` holds `spwn-window` and `Spwn Engine` (the same signed darwin-arm64 binary install.sh ships, named for Activity Monitor, ADR-073); `Contents/Library/LaunchAgents/com.jterrazz.spwn.{engine,agent}.plist`, the app's own labels, name it by `BundleProgram` and are registered by `SMAppService.agent(plistName:)`; the Swift menu bar is nested at `Contents/Library/LoginItems/Station.app` (LSUIElement, its bundle id kept until the rename); `~/.local/bin/{station,spwn}` are links into the bundle. Agents inside a bundle are associated with it with no key and no team match. Measured with ad hoc code (ADR-073): one "Spwn" in Login Items and, with the nested engine signed with the bundle's own identifier, one Local Network row named Spwn, asked once (`com.jterrazz.station` or a default gives two); one Dock icon and one notarized zip are expected. No GUI touches the local network: the window and the menu bar speak to the engine over its unix socket alone, and in mode A launchd starts the engine, never the window as its child (`apps/desktop/src-tauri/src/local_core.rs:251` is the fallback that would break it).
- **Mode B, a headless Mac (install.sh alone), amd64 and macOS 14 kept**: today's bare binary and plists with `ProgramArguments`, minus `AssociatedBundleIdentifiers`, since a Mac that holds Spwn.app is mode A. Its Login Items row reads the certificate's organisation name (proved, 6dce2c34d) and its Local Network row the executable's file name (proved, ADR-073). Linux is outside: the background adapters are `launchd` and `fake` alone (`apps/engine/background/adapter/`).
- **The menu bar merges into Spwn.app** (an idea the owner approved): a nested login item, switched by `SMAppService.loginItem(identifier:)` from a window Settings tab, "Show in menu bar". It keeps the Swift panel, ADRs 021, 036 and 069 and the stories suite (ADR-026). A Tauri tray (loses the native panel and its rig) and a Swift executable in `Contents/MacOS` (`Bundle.main` becomes Spwn's) were weighed and lost.
- **B to A is silent**, at the app's first launch, when it finds a bare job (the hello's `executable` outside the bundle, or `~/Library/LaunchAgents/com.jterrazz.station.engine.plist`): boot out the agent's job then the engine's, delete both plists, register the bundled agents, point `~/.local/bin` into the bundle, rewrite `station/installed.json`, run Station.app's `--unregister-login-item` and delete `~/Applications/Station.app` (`catalog/packages/system-utilities/station-menubar.yaml`, `removal`). The VMs survive an engine restart and are adopted again (`specs/cli/engine/adopts.spec.yaml`; E3 tier 2 on the Mac mini, 2026-11-21). Nothing is kept ([ADR-072](docs/decisions/072-before-1-0-every-station-runs-the-same-release.md)). A to B: install.sh on a Mac with Spwn.app links into the bundle and writes no job (L6).
- **What only the user can clear**: a deleted Login Items record may stay until macOS's overnight maintenance, and the only reset, `sudo sfltool resetbtm`, resets every third-party item, so it is the owner's call alone and never the updater's. The owner reports whether the owner-name row went overnight. A stale Local Network row has no API: since macOS 27.2 the user selects it and presses (-), which the window's migration panel says once, in words, and never automates. HYPOTHESIS: `tccutil reset` does not cover Local Network (TN3179 names no tccutil).
- **Dev builds never run on a real Mac** without a stable signing identity: a bare build adds a row for each new path or identifier, and an updated ad hoc bundle lost both its job's start and its Local Network grant (ADR-073, HA3).
- **What L0 settled** (2026-11-21, development-macos and the Mac mini, ad hoc probes; the evidence is ADR-073's):
  - proved: H1; HA1, the `~/.local/bin` link included; HA2, a register with no GUI session failing with error 125 while `status` reads `enabled`; HA4 (`kickstart -k`, and `register()` after a `bootout`, never a `bootstrap` of a bundled plist); a nested login item listed under its parent (BTM `Parent Identifier`);
  - half: H2, the "Station" row is the menu bar's rule, but what raised it is unknown; HA5, from Tauri's source alone;
  - refuted: HA3 for ad hoc code, whose update broke the job's start and the grant;
  - one Local Network row in mode A, by signing the nested engine with the bundle's identifier (ad hoc, the mini, macOS 27.0.1);
  - one grant across the modes: no by path, yes by Mach-O UUID once.
  - Open, HYPOTHESIS (2026-11-21): that a Developer-ID update keeps the job and the grant (L2's signed builds settle it), and whether `AssociatedBundleIdentifiers` moves a Developer-ID engine's Local Network attribution (a Developer-ID engine on a Mac with no rule for either app settles it).
- **Lots, in order**:
  - L0, done (2026-11-21): ADR-073, Proposed, and the decisions index. Left to L2: B to A on a Mac with an approved Developer-ID engine, since no signed bundle with a nested agent exists before it.
  - L1, done on `feat/one-spwn-l1-l2` (2026-11-21): `scripts/sign-macos.sh` `sign` signs every Mach-O under `Contents/MacOS` but the main executable with the bundle's own identifier, inside out; an ad hoc `make window` on development-macos gives `Contents/MacOS/Spwn Engine` `Identifier=com.jterrazz.spwn`, and `codesign --verify --deep --strict` passes. Left: the notarized publish dry run on main, below.
  - L2, code done on `feat/one-spwn-l1-l2` (2026-11-21): publish builds station darwin-arm64 before `make window`; the `window` target and `apps/desktop/src-tauri/tauri.conf.json` embed it as `Contents/MacOS/Spwn Engine` with the two `BundleProgram` plists, and `make window` checks its version; the engine's start records the environment in `~/.local/state/station/jobs.json`. Its two mode A gaps closed on the same branch (2026-11-22, ADR-073's "What L2's smoke found"): "installed" is one question of the job manager port (`contract.Jobs.Installed`), which every former reader of the bare plist asks, and the bundle registers the app's own labels, `com.jterrazz.spwn.engine` and `.agent`, so the move needs no reboot. Pending, after the merge: ADR-073's Developer ID gate on the Mac mini, since signing trusts only main and `v*` tags. Then ADR-073's gate, before L3: two Developer-ID builds from a `workflow_dispatch` on main, installed in turn on the Mac mini, the first registered and approved; Local Network must show ONE row named Spwn across both, and the second must start after `unregister()` then `register()` and reach the LAN with no new question. A failure goes back to the owner.
  - L3, the engine picks its mode: `apps/engine/background/adapter/smappservice/` (cgo) when `filepath.EvalSymlinks(os.Executable())` is inside `*.app/Contents/MacOS`, else `launchd`; mode B drops the key (`catalog/launch.go`, `StationApps`); the B to A move in `engineInstaller` (`wire/engine.go`); a restart after an update is `unregister()` then `register()`; whether the job runs is judged by `launchctl print`, never by `SMAppService.status`, which answers only whether it is installed. Proof: `specs/adapters/smappservice/{start-registers,migrate-boots-out-the-bare-jobs,migrate-keeps-vms}.spec.yaml`, `specs/usecases/engine/start/installs-on-a-fresh-mac.spec.yaml` still judging mode B, and two `specs/machine/macos/tier.sh` cases: a mode A install whose dumpbtm parent is `2.com.jterrazz.spwn` with no plist in `~/Library/LaunchAgents`, and B to A with the bare plists gone and the VMs adopted. The adapter, the mode pick, the B to A move, the port's installed answer and the app's labels landed with L2; mode B dropped the key on `feat/one-spwn-l3-l5` (2026-11-22). The three `specs/adapters/smappservice/` documents are not written, since no spec reaches mode A: a world has no bundle (`wire/jobs.go`, `bundle()`), and SMAppService is cgo, off Linux; the adapter's module tests (`background/adapter/smappservice/smappservice_test.go`) and the move's (`wire/jobs_test.go`) stand in, with the ad hoc smoke on a throwaway clone of development-macos. The two `tier.sh` cases are still to write. An ad hoc smoke on a fresh throwaway clone of development-macos (2026-11-22, no reboot, no logout) proved mode B, the move with both jobs running at once under `2.com.jterrazz.spwn`, `station engine start` exiting 0 in mode A, a stop reading "stopped", and an update re-registering both jobs.
  - L4, the menu bar is nested. Landed on `feat/one-spwn-l3-l5` (2026-11-22): `make window` builds the menu bar and nests it at `Contents/Library/LoginItems/Station.app`, publish signs and ships Spwn.app alone, `station-menubar.yaml` and `menubar-install`/`menubar-uninstall` are gone, and the B to A move unregisters and deletes `~/Applications/Station.app` and, when there was one, registers the nested menu bar in its place (`SMAppService.loginItem`, from the engine). Left: the window's switch and the panel's, and the rig's `dock` set, below. The plan was: `apps/menubar/scripts/build.sh` builds into `Spwn.app/Contents/Library/LoginItems/`, publish drops `Station.app.zip`, `catalog/packages/system-utilities/station-menubar.yaml` goes, and the window gains the Settings tab and a Rust command calling `SMAppService.loginItem(identifier:)`. Proof: the menu bar rig's `dock` set (`specs/machine/menubar/run.sh`: one Dock icon, the helper running from inside the bundle), and the window's scenarios and goldens for the tab (`apps/desktop/SCENARIOS.md`). The menu bar tells a stopped engine from one never installed by asking the CLI it already runs, `station engine --json` (`engine.state` reads `not-installed`), which answers from `contract.Jobs.Installed`, so no job file is named in Swift (`apps/menubar/sources/station-kit/engine/EngineInstall.swift`, proved by `EngineInstallTests` and the menu bar rig; mode A with the engine down, on a Mac, is not run).
  - L5, the CLI is a link: the B to A move, which the window's first start runs, writes the `~/.local/bin/station` link (the bundle carries no `spwn`), and `station_of` (`apps/desktop/src-tauri/src/station.rs`) finds the bundle's station first, before the one a running engine names (`feat/one-spwn-l3-l5`, 2026-11-22, proved by its Rust unit test). Left: the `tier.sh` B to A case.
  - L6, in spwn-web: install.sh in mode A links and writes no job, proved by a shell test in that repository.
- **Names**: every name the rename owns stays as it is (the binaries `station` and `spwn`, the bare jobs' labels `com.jterrazz.station.*`, the state folders, the socket, `STATION_*`, the schema ids, Station.app's bundle id and its preferences domain); only where the binaries live and who registers the jobs change, and mode A's two jobs carry the app's own labels, `com.jterrazz.spwn.engine` and `.agent`, so that no record of a bare job matches them (ADR-073). So L1 to L5 land before the command talk ("Station becomes Machines", above), which later renames files inside one bundle, and the menu bar merge depends on L2's layout alone.
- **After alpha.3, on the owner's Mac and the Mac mini**: the owner's Mac lists two Login Items rows for one job, "Spwn" (associated by alpha.3) and the owner's name (the record made before it). HYPOTHESIS (2026-11-21): the old record clears at the next login or overnight. Mode A's migration removes the bare jobs that hold it; until it ships, the owner's report settles it, and `sudo sfltool resetbtm` stays his.

L1 starts once the owner has read ADR-073: mode A gives one Local Network row, if the nested engine carries the bundle's identifier.

## `station uninstall` undoes its own writes

Today removal is a recipe by hand ([Getting started](docs/start/getting-started.md#5-remove-what-it-put-down)), and `station uninstall` is an unknown command (QA, 2026-11-21). The command undoes what station wrote, by its ledger (`ledger.json`) and the Footprint's rows, never by a path list of its own: the engine's two jobs booted out, the plists, Station.app, the `# >>> station >>>` blocks, the keys `init` set in the owner's files, the folders, then its own binary last, and `--keep-data` for the VMs.

- **When it starts**: with U3 of "Install and upgrade for customers", since the versioned layout and `install.json` are what it removes (`~/.local/share/station/versions/<v>/` and the symlink), and `station init` must record its edits in the ledger first, since the keys it sets in `~/.npmrc` and the other three files are not recorded today.
- **Mode A**: with Spwn.app (L2 and L3 of "One Spwn on a Mac"), the jobs are `SMAppService` registrations the app owns, `station engine stop --remove` already unregisters them (or deletes a bare install's plists), and the command calls it first; `~/.local/bin/{station,spwn}` are links it removes, not binaries. Mode B is what the recipe covers today.
- **Not in the installer**: `install.sh` is spwn-web's, so the profile block it writes (`# Added by spwn installer`) is removed by the command from a marker the installer's record names; that record is a spwn-web change, owed first.

## The read model, RM6 and RM7
The engine keeps one read model, whose projections are the menu bar's pages and the window's views ([ADR-070](docs/decisions/070-the-engine-keeps-one-read-model.md)). On main: RM1, the store (0cc22d9a3); RM5, the verdict in `opened` (74b908ec0); RM2, readers keep the store warm and `engine.state` streams it (1054632c4); RM3, every Mac holds every other Mac's `local` stream (dd4447ae2, ADR-072); a Mac's Accounts pending until its sign-ins answer (1395dcd7b); and RM4, the menu bar's pages projected from the store (`engine/feeds`, the rule `feeds-read-no-call`). The contract they landed is the `engine.state` paragraph and the pages paragraph of [`docs/reference/api.md`](docs/reference/api.md), which this item does not repeat. What is left is the window's half, RM6 and RM7.

- **The version stories need a stamped build**: `prints-commit`, `far-on-another-release` and their siblings read the release `make test` stamps (`SPEC_LDFLAGS`, `git describe --long`); a spec binary built by hand without `-ldflags` fails them, and so does one stamped from a dirty tree (`prints-commit`). They are green in `make test` and in the cc loop.
- **RM6 and RM7's first half, on `feat/window-reads-state`**: `engine.state` streams the views `machines.list`, `overview.read`, `overview.read+elsewhere` and `overview.read:<m>` beside the slices (`engine/feeds/views.go`, stories `engine/state/views-equal-calls*` and `resume-another-engine`), and the window reads the first three from one held stream (`StateMirror`, `apps/desktop/src/application/use-cases/mirror.ts`; its record is `apps/desktop/README.md`, "The engine's views are mirrored").
- **RM7, what is left**: a machine's own Overview routed to `overview.read:<m>`, so the store serves no read the engine keeps (ports, sign-ins and Awake are views since `feat/views-ports-accounts-awake`); another Mac's `awake.read:<mac>` differs from its call in `NeverSince` and the verdict it moves (story `engine/state/views-equal-calls-awake`; HYPOTHESIS, 2026-11-22: the far Mac's `awake:` slice carries no `NeverSince`, which its engine's call reads from its preferences; reading that slice's value in the story settles it); a VM's sign-ins whose reading never answered sends no view, so its tab stays a skeleton where the call drew its refusal (HYPOTHESIS, 2026-11-22: a stopped VM's tab is the one case; a page spec over `leftovers` settles it); the engine's `fit` from `opened` replacing `judged` and the hello's page check in `use-cases/reach.ts`; `FRESH_EVERY` and the 10 s hello deleted; the last snapshot kept on disk; views projected once per revision (each held stream projects its own today); a gesture's `rereads` of a view asking the engine to read it again rather than waiting for its readers' tick (HYPOTHESIS, 2026-11-22: an Overview need fixed from the window leaves the list only at the next overview read; one fix in the real app settles it). Proof still owed: the launch capture of the first frame at 0.8 s and the `--fault=engine.hello` capture of the no-engine panel ([ADR-071](docs/decisions/071-the-window-mirrors-engine-kept-state-never-blank.md)).
- **What RM6 and RM7 read** (types: `engine/state.go` and `engine/fit.go`; schema `packages/api/schemas/calls/engine.state.json`, TS in `packages/api/index.d.ts`):
  - The window subscribes once, over the engine socket or `guest serve --stdio` (which relays the engine's stream; no engine running is a DownError): `{"v":2,"id":"1","watch":"engine.state","in":{"keys":["local"],"revision":1,"hold":true}}`, with no `pace` (another Mac's engine sends `idle` while no client is attached to it). It applies on a Mac only; a server or a workspace refuses `section.not-for-this-machine`.
  - Frames: `opened` (with `fit`, `epoch` and `rev`), then `snapshot` or `changed`, then a `tick` every 15 s. A misfit (`engine-older` or `app-older`, strict equality, ADR-072) ends the watch after `opened`, with no slice sent; its words are the menu bar's.
  - `changed` carries only the keys whose hash or state moved, coalesced over 250 ms, plus `gone`. A slice whose hash did not move comes without `value`: the window keeps the value it holds.
  - 45 s with no frame means the stream is lost: the window resubscribes with `since` `{epoch, rev}`, and gets `changed` when that revision is inside the last 256, else `snapshot`.
  - A slice carries `state` (`fresh`, `stale`, `unread`), `why` (`reading`, `late`, `unreachable`, `far-too-old` or an error code) and `read_at`. Values hold facts only: `overview:<m>` is `{Summary, Boot}`, so uptime (now minus `Boot`) and the head line are computed at draw time. The value types are `mac` machines.MacView, `ports` ports.MacPorts, `overview` overview.Kept, `awake` awake.Report (`Now` zeroed), `activity` activity.Report and `signins` accounts.ProgramsRead; measured in `two-macs`, a Mac's slices run 0.7 to 6.2 KB each.
  - The fleet is asked by far keys: `["mac:*","overview:*",...]` matches every other Mac's slices, which `local` never does, and a far slice's value has its family's own type. A far Mac's `far-too-old` is drawn as "update station on <mac>".
  - In a world (the stories, the recordings) the readers are lazy: they start at the first `engine.state` ask (`readers.Options.Lazy`), not at the engine's start, and a stepped world engine waits once, 5 s at most, for each far Mac's first snapshot, so the first frame already holds them.
- **Hypotheses**: a held far watch inside the world does not stall the clock's leap (RM3's pushed story under `-race` in the cold gate); every field holding a read instant in a slice value is listed (RM2's `unchanged-world-same-revision` story); the folded activity slice stays a few KB (sized in RM2's story); a held watch's tick does not keep a far Mac awake (the owner's Macs alone).
- **Open, the owner's**: whether `apps`, `settings` and `disk` become slices kept forever (brew, apt and settings reads every 10 min on every machine; recommended, a separate ADR once measured, the window keeping those calls meanwhile); whether `station machines` reads the engine's view when an engine runs (not proposed).

Starts: the rest of RM7 once `feat/window-reads-state` is on main.

## The window after alpha.3
What the window's session left open when it closed (2026-11-21). Its record is `apps/desktop/README.md`, `apps/desktop/SCENARIOS.md` and ADR-071.

- **The owner's eye on alpha.3**, in the real macOS app: the first frame at launch, the skeletons, the stale mark ("as of 10:42"), the incompatible-version panel, the Engine page's Update, the first update and relaunch. Starts when the owner opens alpha.3.
- **The Dock reopen**: HYPOTHESIS (2026-11-21) that a click on the Dock icon while no window is on screen shows the window again (`RunEvent::Reopen`, as `apps/desktop/README.md` states); one click after closing settles it.
- **Scenario A7**, the window shown at its first data (`apps/desktop/SCENARIOS.md`), stays `no`: no bench runs the native window. Starts with a bench that captures the Tauri window on development-macos.
- **The window mirrors `engine.state`**: RM7, in the read model's item above.
- **Waiting on the engine or the owner**: the input `Default` (D15, D19), the second half of D6, the `engineRestarts` sentence, one press that updates the engine and the window (mode A of "One Spwn on a Mac" makes it one update), the update row in the quiet window, and where `machines.server-needs-name` is drawn (no recording has it yet).

## studio-infrastructure on a station server: the spwn lots
The owner's vision (accepted 2026-11-20, its five recommendations on 2026-11-21): the cluster's machine is a Debian 13 server that station builds from its Machinefile on the Mac mini; studio-infrastructure's CI is the only deployer of what runs on it, over Tailscale and ssh; secrets stay with the deployer (that repository's ADR-015); the Mac holds station and its vault alone. No GitOps, no secret in the VM, and the VM never clones the repository. The Machinefile owns the base apt, UTC, unattended-upgrades, sshd's hardening, the CI's public key, systemd-resolved and its upstreams, the sysctls and modprobe blacklist, and Tailscale joined by a vault grant; it changes only on the owner's verb, never by a night or drift job. studio-infrastructure's `TODO.md` holds its own lots and the cutover; these are this repository's, each with its stories, a tier-2 case and a cold `make check`.

- **S1, base-debian**: `catalog/machines/base-debian.machine.yaml` (`{os: debian, version: '13', arch: arm64}`, the upstream pinned by digest), a Debian recipe in `catalog.RecipeFor` (`apps/engine/catalog/images.go:142-150`), `os: debian` counted as Linux (`catalog/convert.go:6-9`), `family: debian` on the apt packages that hold, and `docs/reference/images.md`, which calls it a proposal. Proof: stories, a tier-2 `image-build` case on base-debian. ADR-051 already decides it.
- **S2, a server from a Machinefile**: lift the refusal (`surface/cli/machines/create/command.go:188-193`, `machines/calls/create.go:82-83`, `machines/template.go:76-78`, `wire/machines_templates.go:44`); the record keeps the server's Machinefile; a server runs the image phase, then a first-start phase (`linux-firstboot.sh`: hostname, machine-id, growpart; the fstab mounts; `run.boot`); `station upgrade --on <server>` applies its Machinefile; `--from` takes a path; a server refuses `vm.lifetime` and `network: bridged`; night and drift keep skipping servers (`engine/process/drift.go:25,34`). Proof: the two `machines-server-template.spec.yaml` stories (`specs/usecases/machines/new/`, `create-workspace/`) turn from refused to made, a story per refusal, the CLI goldens, `words.json` and the schema by `make generate`, and a tier-2 `create-server-from` case (sized, check passing, `upgrade --on` applying one change).
- **S3, Linux machine settings**: the `file` method writes root-owned `/etc` destinations (mode, owner, a reload command), so each concern is a setting with drift and undo; built-in packages taking their values from the Machinefile: `sysctl`, `modprobe-blacklist` (it creates `/etc/modprobe.d`, absent on a bare trixie), `timezone`, `unattended-upgrades` (one run at first start), `resolver` (systemd-resolved, an upstream drop-in, the stub link; k3s cannot start a pod without `/run/systemd/resolve/resolv.conf`) and `sshd-hardening` (a `70-*.conf`, keys only, station's forwarding and keepalive kept). HYPOTHESIS (2026-11-20): today's `file` decoder refuses an absolute path; read `catalog/decode.go` before sizing the lot. Proof: stories per outcome (applied, drift, undo, refused outside `/etc`), a tier-2 case reading `timedatectl`, `sysctl`, `sshd -T`, `resolvectl`, `/etc/modprobe.d` and the unattended stamp.
- **S4, an authorized public key**: an `authorized-key` setting (`user`, `key`) writing a block in `~user/.ssh/authorized_keys` (`/root` for root) and keeping a person's lines (ADR-043's principle). Proof: stories (added, kept beside a hand line, removed), a tier-2 `ssh -i <test key> root@<server> true`.
- **S5, Tailscale on Linux and a grant to a server**: a Debian method in `remote-access/tailscale.yaml` (the pkgs.tailscale.com apt repository, tailscaled enabled), a `tailscale` delivery (`sudo tailscale up --auth-key="$(cat)" --hostname=<vm>`), `GrantToServer` lifted for that delivery alone (`accounts/adapter/keychain/grants.go:176-178` and the fake). The key is one-off, pre-approved, tagged `tag:cluster`, expiring in a day. Proof: stories (granted to a server, still refused there for gh, npm, claude and wrangler, the value in no log, ADR-048), a tier-2 case against the stub tailnet, and a `server-facts` case once S2 lands: an fsync-per-write measure on the data disk and on the boot disk (whether studio-infrastructure's write-through unit goes), and `ip -br a; ip route` (the guard's values).
- **S6, a data disk** (the owner, 2026-11-21, replacing a share of the Mac's folder): a server's Machinefile declares a second disk image (tart `--disk`), ext4, at `/srv/data` in the guest (k3s's local-path at `/srv/data/k3s`), kept across a rebuild of the boot disk. It needs its ADR, amending 044 and 033. Since the rehearsal's ext4 enforced ownership (five stateful workloads crash-looped on `root 0755` volumes), studio-infrastructure's ownership lot applies.
- **The ADRs, written before S2**: amend [ADR-017](docs/decisions/017-servers-and-workspaces.md) (a server is made from its Machinefile, set up at its creation and on the owner's `upgrade --on`; accounts stay refused but the `tailscale` delivery), [ADR-044](docs/decisions/044-an-image-is-baked-from-its-machinefile.md) (a server runs the image phase and a first-start phase), [ADR-033](docs/decisions/033-a-machine-is-declared-by-its-machinefile.md) (`kind: server` is a usable Machinefile) and [ADR-049](docs/decisions/049-a-vm-signs-itself-in-the-vault-grants-the-rest.md) (the Tailscale auth key is the one grant a server takes, delivered once).
- **Sizing**: 16 CPU and 30 GB on the new Mac mini M5 Pro, the production target (the owner, 2026-11-21). The rehearsal (2026-11-20, a Debian 13 arm64 stand-in on development-linux with fake secrets) measured about 2.6 GiB of working set, a lower bound, and 12 GB of disk.

Starts with S1 under the owner's go of 2026-11-20 for the plan after alpha.2; S2 after the ADR amendments, S6 with S2, S5's facts case once S2 lands.

## The train, the faster stories and the warm lanes
[ADR-066](docs/decisions/066-a-commit-owes-that-it-builds-and-lints-a-lot-owes-the-gate.md) and [ADR-067](docs/decisions/067-one-train-appended-by-makers-proved-at-its-tip.md) set the cheap per-commit tier, the full gate per lot tip and the train. The scripts that run the tiers are in `scripts/gates/` ([Test](docs/guides/test.md), "The gate scripts"); what remains is the train's own scripts and the speed work of the runs audit, each a project of its own.

- **The train in scripts** (L5): `train append`, `prove` and `bisect`, the integrator's state two refs. Proof: the next lots land through it, and a staged red is bisected to its lot in four runs.
- **Stories wait on the world, not on real time** (T4): the six subscribe documents of `specs/cli/engine`, `crash-keeps` and the two `adapters/ssh/ports-*` limits wait on what they assert. Left: the documents whose claim is an absence over a real tick (`adapters/pmset/{display-request,clamshell-resets,no-file}`, `cli/engine/{certificate-serve-off,logs-guest-errors,adopts}`, `cli/guest/docker-steady`, `usecases/overview/read/vm-being-created*`) and the 3 s before `subscribe-menubar`'s watch, each waiting for an engine reading that says its tick or This Mac's first read is done; then the `/bin/sleep 0.1` socket polls. Proof: each golden unchanged byte for byte, 100 runs beside two lanes.
- **Warm lanes** (T5, its ADR still to write): worktrees at fixed paths on the VM, a bare mirror, `git checkout --detach <sha>` and a clean that keeps `.artifacts/` and `node_modules`, so the recorder's key, turbo and Go's caches hit; the cold gate stays a fresh clone. Proof: a docs-only commit in a lane prints the recorder's "skipped, key" and turbo's "cached".
- **Stories carried by content** (T6): the key is the world build's digest (`go build -trimpath -tags world` with a fixed version gave one sha256 from two clone paths, measured) plus the digest of `specs/`. HYPOTHESIS (2026-11-20): the stories read only that binary and `specs/`; one `strace -f -e trace=openat` of a full run settles it. Proof: a docs-only commit replays, an engine commit runs.
- **Pre-warmed tart images** (T8) for the darwin build and `menubar-check`: the base carries the pinned Go, a module and build cache and the SwiftPM scratch folder of a recent main. HYPOTHESIS (2026-11-20): darwin from 83 s to about 35 s, the menu bar check from 190 s to about 60 s; timestamps per step before and after settle it. The release workflow keeps building cold.
- **A third story slot** (L7): three slots and eight workers measured at an exclusive gate. Its precondition, the recorder's determinism, was met on 2026-11-21 (20 forced records at load 37 to 40, one hash).

Starts with the train scripts; T4 to T8 and L7 after.

## The old goldens' last copies on GitHub
The history rewrite (S5) ran on 2026-11-21: `main` is `e82228f9d`, both
alpha tags were re-pointed, and the pack fell from 614 to 173 MiB. GitHub
still keeps `refs/pull/26/head`, `27/head` and `28/head` on the old history,
read only, with cached views of old commits by sha, and the fork
`OpenAgentSystem/spawn` shares the network's object store, so the old
goldens' blobs stay on GitHub until its Support collects them. Starts with
the owner's ticket on GitHub's "remove sensitive data" path, naming the
repository, the rewritten refs, and the pull request refs and cached views
to collect.

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
Today the public `install.sh` (spwn-web's, served at spwn.sh) is the only install and its re-run the only self-upgrade ([ADR-041](docs/decisions/041-the-lock-is-what-apply-installs.md)): no version folder, no rollback, a checksum from the same origin as the binary (integrity, not authenticity). The design below makes one product version stamp every piece (CLI and engine, guest binary, Station.app, the window, the pinned dependencies), released together. Each lot is one maker, in order.

- **Two doors, one layout**: `curl -fsSL <domain>/install.sh | sh` (the primary door) and a Homebrew cask in `jterrazz/homebrew-station` (`auto_updates true`, so `brew upgrade` leaves it). Both land `~/.local/share/station/versions/<v>/` (`bin/station`, `guest/station-linux-<arch>`, both apps), `~/.local/bin/station` as a symlink flipped atomically, and a receipt `install.json` (`channel: script|brew|dev`). The script unpacks into `<v>.partial` and renames; a `make provision` build is `dev` and never updates. On Linux the script installs the binary only, and inside a VM its Mac's engine owns it.
- **A signed manifest**: each release ships `manifest.json` signed by an ed25519 key whose public half is embedded, and a stable `latest.json`. The script verifies the signature, then each asset's sha256. An older binary refuses an unknown major and says to reinstall. Customers also need a Developer ID, the hardened runtime and notarisation; the cask needs them to pass Gatekeeper.
- **The update check**: the engine reads GitHub's latest release every 6 h on a Mac (at start, 30 min after a refused ask) into `~/.cache/station/release.json`, and `engine.read-release` answers it ([ADR-059](docs/decisions/059-the-latest-release-is-read-by-two-calls.md)); still to build: the signed `latest.json` in its place, on wake with jitter. A command reads that cache with no network, and prints at most one stderr line a day per version, on a TTY only. Silent for `--json`, a pipe, `CI`, a VM guest and the `dev` channel. Offline keeps the last cache; after 14 days Needs you says `updates not checked since <date>`. The menu bar and the window show one upkeep row the badge never counts. Opt-outs: `preferences.json` `updates: {check, apply: night|ask|never}`, `STATION_NO_UPDATE_CHECK=1` (check and line), `STATION_DISABLE_UPDATES=1` (every path, `upgrade --self` included).
- **The engine is the one updater**; no Sparkle and no Tauri updater (a second channel that cannot move the engine or the guests, and lockstep keeps `page.Revision` compatible). It runs station's own step first in the 03:00 night ([ADR-042](docs/decisions/042-versions-are-semver-ranges-applied-nightly.md)), or on `station upgrade --self`, in this order, each with its health gate and rollback:
  1. Engine, VMs running: verify the manifest and assets, snapshot the stores, take the machine's turn, write `runs/<vm>.json v1`, flip the symlink, boot the engine's job out and in (`launchctl bootout`, then `bootstrap`). The VMs survive and `Adopt` re-attaches them (`specs/cli/engine/adopts.spec.yaml`). Gate in 60 s: hello answers the new version, every run adopted with its pid, each guest reconnected. A failed gate or three exits in 5 min flips back, restores the snapshot only if a store's version moved, records `update.rolled-back <v>` and skips that version.
  2. Station.app: the catalog script's rename lifted into the engine, then `station.updated`; the app relaunches unless its panel is open. Rollback restores `.old`.
  3. The window: `bundle.active` on and notarised, swapped the same way, "Restart to update" while open.
  4. The guest binary, over the engine's ssh into `/usr/local/bin/station.new`, sha256 checked in the guest, `station.prev` kept, hello gate in 20 s. A stopped VM is pushed at its next boot, before the boot phase. The guest's own `upgrade --self` refuses.
  5. Images: `rebuild waiting` is said, never done unasked (`images.rebuild: ask`, `night` opt-in); two versions are kept.
  6. Dependencies: pinned in the manifest with a sha256, into `deps/<name>/<version>/` behind a `current` symlink, moved only with a station release. A running VM keeps its tart (its wrapper names the versioned path); prune only what no `runs/*.json` names. The unchecked installers (Homebrew, mise.run, herdr) become a `script` method with `url` and `sha256`.
- **Version skew**: CLI and engine are one binary (skew only between flip and restart; after 60 s the CLI says to run `station engine start`). Until 1.0 a guest on another release than its Mac's is refused (ADR-072), so its image is rebuilt at each release. The menu bar and the window are lockstep, each drawing one `page` revision. The engine reads `runs/<vm>.json` of N-1 and N+1, and a file it cannot read is `foreign`: never killed, never adopted, shown. Stores expand then contract, a breaking migration spanning two releases, since a store refuses a newer file today.
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
  - U4: the update check: fetch and cache are done (the engine's 6 h job, `engine.read-release`); the line, the silences and the preferences remain.
  - U5: `station upgrade --self` and the night's first step, with download, verify, snapshot, flip, restart, gate, rollback. It carries the self-replace risk: the release asset with its checksum, replaced by a rename, then the engine restarted (`apps/engine/images/calls/binary.go`'s download lifted to a shared adapter). `station upgrade` today updates what the Machinefile carries, not station.
  - U6: the guest push, at upgrade and at boot.
  - U7: the menu bar's lockstep swap in the engine, the `station.updated` event, the relaunch.
  - U8: dependencies pinned in the manifest (tart's version and sha256 first); the installers with sha256; the launcher question above.
  - U9: `Hypervisor.Capabilities()` and the per-VM `hypervisor` field.
  - U10: the install of the window: publish.yaml signs, notarises and attaches `Spwn_darwin_arm64.zip`, and nothing puts it on a Mac yet.
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
not an option the owner has accepted).

## Machine test tiers: the nightly
Tiers 2 and 3 are built (`make e2e-machine`, [Test](docs/guides/test.md#the-machine-specs)).
What waits is tier 3 as a nightly on the Mac mini, once a tart VM may run
there.

## The Mac mini as an always-on Mac
Installing station there is the owner's go, never a lot's. The production
Mac mini is the new M5 Pro (64 GB, 1 TB); the old one is the test bench
until the cutover, carrying the latest Spwn.app and menu bar, and is erased
after it (the owner, 2026-11-21). On the new one: `station init --name
mac-mini`, the Always-on Mac settings, the infrastructure moved from
OrbStack to a station server ("studio-infrastructure on a station server",
above), images built there. Ask the owner before any test on it.

## The menu bar's stories left after v4
Fourteen gap ids wait for a design before they are stories (STORIES.md,
"Gaps"). Each starts as a lot of its own once the owner asks for it.

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
Three gaps found setting up the owner's new Mac and its first workspace, `os`.

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
- The architect daemon keeps the image's fixed uid, while a world's `spwn` user follows the host user's on Linux (`FollowHostUser` in `packages/architect/spawn.go`, [worlds](../docs/concepts/worlds.md) names the architect container as the gap). The daemon (`StartDaemonWithOpts`, `packages/architect/daemon.go`) bind-mounts the host's spwn folder, so on a Linux host whose user is not the image's uid, what it writes there is owned by another uid than the host user. Not a small change: a remap after `Start` (as a world gets it) misses the process the entrypoint already runs, so the uid has to be set before the daemon starts, in `catalog/architect/tools/architect/files/entrypoint.sh` (which runs first, as root, and fixes the docker socket's group already) from a host uid and gid the daemon passes in its environment, and the image's user model (`architect` in `packages/architect/build.go`, home `/home/spwn/.spwn`) is to be read first. Starts with a Linux host that runs the daemon as a non-root user and sees root-or-foreign-owned files in its spwn folder; proved by running the daemon on the development VM and listing the owners of the folder it wrote.
- The tier harness runs `/usr/bin/timeout` on the Mac (`specs/machine/linux/tier.sh:83,89`), which macOS lacks, and links the real `tart.app` (`:29`), where tier 2 on the Mac mini needed a copy: E3's tier-2 run of 2026-11-21 passed only with a perl alarm shim in place of `timeout` and `cp -R`. Starts with the next tier-2 run on a Mac: a timeout the harness carries for macOS, and the copy.
- `walk` (`apps/engine/engine/process/keep.go:128`) reads the result of the check `keep` runs in a goroutine right after starting it, so a server's uptime record starts on the check's tick or on the next one, by scheduling; the worlds pin it by seeding `stores.uptime.infra.since`. Starts with the next change to keep: walk waits for the check it started.
- An engine upgraded by spwn-web's `public/install.sh` keeps running the old binary, since the script restarts no job (it makes no `launchctl` call), and on the owner's Mac after alpha.2 a `launchctl kickstart -k` was refused, `OS_REASON_CODESIGNING` ("needs LWCR update"), until the binary was replaced on a fresh inode (`cp` then `mv`); the Mac mini's upgrade took one `kickstart -k`. The engine's own restart boots its job out and in since e562644ec. HYPOTHESIS (2026-11-21): launchd's recorded code requirement for a job loaded from an ad hoc build refuses a Developer-ID binary at the same inode; settled on a VM by an engine job from a local build, then install.sh of a release, then `kickstart -k`. Starts with U5 (the engine as its own updater) or mode B of "One Spwn on a Mac", whichever lands first: the upgrade restarts the job by bootout and bootstrap.
- `go work sync` on main rewrites seven `go.sum` files: HYPOTHESIS (2026-11-21, seen by the uid lot), settled by `go work sync && git status --porcelain` at main's tip on development-linux; if it holds, one commit syncs them, as a866d0102 did for `specs/`.
