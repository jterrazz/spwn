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
- **Where they live**: a git repository of the owner's, so history is the
  undo. The engine reads it from one path, and every Mac's engine reads the
  same commit.
- **Plan, then apply**: `station plan` prints the difference between the
  blueprint and the machine (to install, to set, extra), and `station apply`
  does it, idempotent and check-first like `setup`. Removing what a blueprint
  does not list is a separate flag, never the default.
- **Drift**: the engine compares on its own and the menu bar shows "3 apps
  missing, 1 extra" on the machine's card, with Apply.
- **Images**: a workspace role's blueprint is baked into its image by the
  recipe, so a new VM is ready at boot instead of installing on first start.
- **Capture**: `station blueprint capture` writes a blueprint from a machine
  as it stands, so the first file comes from the old MacBook's inventory (its
  Brewfile and `apps.md`, in the migration folder of 2026-11-10).
- **The minimal host**: the owner's Mac keeps station, tart, Tailscale, Claude
  Code, git, gh, gpg, Ghostty and Herdr, and lets Claude iterate on station
  from the host. Everything else installs in workspaces.
- **The visual side**: a Blueprints page per role, with the catalogue as
  toggles and values in forms. It writes the files, which stay the source of
  truth, and it is designed with the owner (prototype, then stories) before any
  lot.

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

## Choose the secret manager
Bitwarden Secrets Manager or 1Password service accounts, as a second adapter of
`accounts/contract.Vault`. It becomes the home of the secrets shared with a VM
beyond `session:*` (`operator:*` tokens scoped per VM). Starts when a VM first
needs a token that is not a browser session.

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
upgraded: `station setup --name mac-mini` there, the Always-on Mac settings,
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

## Long-running agents
Hermes and OpenClaw as long-running VMs on an always-on Mac. Not designed: the
owner still starts them by hand. Their gateway rows were dropped from the
Overview until that design, which may be a machine kind of its own.

## Network isolation
Tart Softnet for VMs that need no LAN. Until then a VM behind its Mac reaches
what its Mac binds on every interface, and the other VMs of its Mac (sshd,
Docker's ports), whatever the Tailscale policy says; the Ports section shows
each as Public when it is.

## Smaller follow-ups
- A server's traffic on its card: a metrics URL a server declares (Prometheus text, read over the tailnet from its Mac), its requests per minute beside the latency ([ADR-028](docs/decisions/028-a-servers-card-shows-latency-not-traffic.md)). Starts when the owner asks for it.
- An image registry (private ghcr) once two Macs build the same image.
- Memory soak: a Linux VM 8 h under load, the Mac's RSS of the VM process.
- `machines edit <vm> --permanent`: turn a throwaway workspace permanent (rewrite its record; the engine reads it).
- HTTPS mode: if a Mac's `tailscale serve` refuses arbitrary https ports, the default of the `ports` group's `https` becomes `path`.
- Remove the old Tailscale leftovers by hand when wanted: the `tag:station-vm` tag owner, the VM-join OAuth client, each Mac's userspace node (admin console), `brew uninstall tailscale` (the formula; the app replaces it).
- The poweroff intent over a FIFO instead of a console file, if the macOS guest's console differs.
- A second `Hypervisor` adapter (Lima, Apple `container`, Virtualization.framework direct) if tart's licence or owner turns bad.
- A second `Notifier` adapter (ntfy, to the phone), and a `Jobs` adapter over systemd for a Linux machine that runs an engine.
- Herdr: `machines create` could run `hc machine add <name> [--temp]`.
