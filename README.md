# station

A single CLI to set up and run a Mac, its VMs and your other Macs: the apps and agent skills, the settings, the disk, the ports and who reaches them, the accounts shared with a VM, Awake, and the servers and workspaces where development lives. No sudo required for the install.

## Install

**Fresh machine** (no Go needed):

```sh
xcode-select --install
curl -fsSL https://raw.githubusercontent.com/jterrazz/studio-station/main/scripts/install.sh | sh
exec zsh
station init --name macbook
```

The binary lives at `~/.local/bin/station`; its files follow the XDG folders. The first steps after it are [Getting started](docs/start/getting-started.md); what it writes is [Footprint](docs/reference/footprint.md), and the release it comes from [Release](docs/guides/release.md).

## What it does

```sh
station                 # the same as station overview, what needs you first, then each section

# Machines
station machines        # All machines: every Mac, its servers and its workspaces; create, start, stop, delete
station agents          # every agent of every workspace, by its name; start, reply, attach, merge, stop

# On a machine (--on <machine>, or station use <machine>, else This Mac)
station overview        # this machine at a glance: what needs you first, then each section
station activity        # CPU, memory, disk, network, thermal, battery, the busiest processes
station apps            # install, update, remove apps and agent skills
station settings        # what station sets up and checks: applied, changed since, off here
station disk            # what each app keeps: free what rebuilds, erase data, move what is misplaced
station ports           # what listens, Private or Public, and what your VMs share on Tailscale
station accounts        # who is signed in, and the accounts shared with VMs: never a token
station background      # the background tasks: failing, orphans, station's, the apps'
station awake           # keep a Mac from sleeping: every Mac's mode, a keep-awake for a while

# A Mac's images and engine
station images          # the VM images of every Mac
station engine          # this Mac's engine: running, its version, the VMs it runs; start, stop

# This terminal
station use linux-1     # every later command of this terminal runs on linux-1

# Station itself
station init            # make this machine known: its name, the shell block, the XDG folders
```

Every command is in [Commands](docs/reference/commands.md); `--on <machine>` runs any of them elsewhere.

The menu bar app, `apps/menubar`, shows All machines, Needs you, Activity and Awake from this Mac's engine. `station init` never installs it: `make menubar-install` copies it to `~/Applications`, and [Develop](docs/guides/develop.md) holds its targets.

## Documentation

The corpus lives in [`docs/`](docs/README.md); start at its map, which lists every page. The words are in the [Glossary](docs/concepts/glossary.md).

For agents: start at [`CLAUDE.md`](CLAUDE.md). The [`skills/`](skills/) directory ships the shared skills: `studio-toolbelt` (the machine's roster of apps, generated from the catalogue), `studio-stack`, `studio-new-project` and `studio-repo-structure`. What is left for later is [TODO.md](TODO.md).

## License

MIT © [Jean-Baptiste Terrazzoni](https://github.com/jterrazz)
