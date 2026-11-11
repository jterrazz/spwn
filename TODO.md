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
  Today station never touches `~/.claude/settings.json` on a Mac (only
  `agent-hooks`, in VMs), so the new MacBook's were set by hand: session
  retention (`cleanupPeriodDays: 365`, against a default that erased the old
  Mac's sessions past 30 days), the auto-compact window, notifications, the
  theme, the statusline and the `no-local-poll` hook (rtk's hook only where
  rtk is installed). A blueprint layer declares them and `apply` writes them,
  the owner's keys beside station's. Open: the Claude theme is copied today by
  studio-os's `apps/herdr` install, so one of the two owns it; and whether
  moving a checkout's sessions and memory to its new path (done by hand on
  2026-11-10) is a station gesture.
- **apply installs the Machinefile's choice (for the manager lots, L3.x)**:
  the resolution chooses an item's method (an entry's `manager`, the
  system's priority) and its version, and `explain` prints them, but
  `machinefile.apply` hands `apps.install` the name alone (`InstallIn` has no
  method or version), and the shelf installs by the first method of the
  catalogue that fits this machine (`Shelf.install`, `InstallOn`). So a
  `{ name: node, manager: mise, version: 22 }` is installed however the
  catalogue would, unpinned, and the plan's `method` column can name another
  manager than the one that runs. `apps.install` takes the step's method and
  version, and apply passes them, once the manager lots give apt and
  github-release an installer (L2.3's re-verify, 2026-11-11).
- **git's identity**: no setting writes `user.name` and `user.email`; Accounts
  only reports them unset, and `git-signing` reads them from the preferences
  without writing them. The new MacBook's were set by hand (2026-11-10); a
  blueprint names them.

It replaces the idea of `station apps install --all`. Starts once the new
MacBook runs station v3 and the owner opens the design.

## The next minor release: the v2 words go
The hidden aliases (`status`, `start`, `stop`, `tools`, `secrets`, `config`,
`storage`, `run`, `probe`, `--always-on`) and `station.status/v1` are kept for
one release ([Commands](docs/reference/commands.md#hidden-aliases)). The next
minor removes them, with the `machines supervise` retirement shim and the
retired preference names (`here`, `hosts`, `services`, `lend_list`), once
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
so sharing credentials between agents, VMs and Macs is one gesture of the app:
a credential is shared with a VM or an agent for a term, scoped
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
Seven stories A2 designed are in STORIES.md but not built: the Agents
fleet page (`agents.list`, `agents.empty`), the New agent sheet
(`sheet.new-agent`, `sheet.new-agent.new`) and the agent's menus
(`menu.context.agent`, `menu.page.agent`, `menu.new-agent.on`). Fourteen
gap ids wait for a design before they are stories (STORIES.md, "Gaps").
`share_ended` is a notice kind the HUD knows and no engine job tells yet:
the take-back of a copy at its term is where it would be told. Each starts
as a lot of its own once the owner asks for it.

## Permanent agents, and what each machine holds of them
The owner's question (2026-11-10): Herdr runs the agents, so every machine
that carries Herdr (the app on the owner's MacBook, herdr-core on the Mac
mini and in each VM) potentially runs its own permanent agents, with their
own runtime, sessions and state. That opens what one machine knows of
another's agents, and how the differences between machines are seen and
reconciled: which agents run where, which sessions and memory they keep,
which accounts each machine holds or was lent (accounts diverge the same
way). The menu bar is where these differences are seen (All machines,
Agents, Accounts), and a Machinefile may one day declare a machine's
permanent agents. Not designed; decided with the owner before any lot.
Hermes and OpenClaw, which the owner still starts by hand on an always-on
Mac, are the first cases; their gateway rows were dropped from the Overview
until this design, which may be a machine kind of its own.

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
- A server's traffic on its card: a metrics URL a server declares (Prometheus text, read over the tailnet from its Mac), its requests per minute beside the latency ([ADR-028](docs/decisions/028-a-servers-card-shows-latency-not-traffic.md)). Starts when the owner asks for it.
- An image registry (private ghcr) once two Macs build the same image.
- Memory soak: a Linux VM 8 h under load, the Mac's RSS of the VM process.
- `machines edit <vm> --persistent`: turn an ephemeral workspace persistent (rewrite its record; the engine reads it).
- HTTPS mode: if a Mac's `tailscale serve` refuses arbitrary https ports, the default of the `ports` group's `https` becomes `path`.
- Remove the old Tailscale leftovers by hand when wanted: the `tag:station-vm` tag owner, the VM-join OAuth client, each Mac's userspace node (admin console), `brew uninstall tailscale` (the formula; the app replaces it).
- The poweroff intent over a FIFO instead of a console file, if the macOS guest's console differs.
- A second `Hypervisor` adapter (Lima, Apple `container`, Virtualization.framework direct) if tart's licence or owner turns bad.
- A second `Notifier` adapter (ntfy, to the phone), and a `Jobs` adapter over systemd for a Linux machine that runs an engine.
- Herdr: `machines create` could run `hc machine add <name> [--temp]`.
- `ansible` installs on Linux by `mise: pipx:ansible` and names no `requires`, unlike `markitdown` and `browser-use` (`requires: [uv]`), so neither the resolution nor `apps install` brings uv or pipx before it. Starts with the answer, read on a Linux VM: does `mise install pipx:ansible` work with neither uv nor pipx there? If not, `ansible` requires what it leans on, as the other two do.
