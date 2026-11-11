# The manual of station

`station` is the CLI that sets up and runs a Mac, its VMs and your other Macs: it installs apps and agent skills from a catalogue, applies and checks the machine's settings, keeps servers alive and workspaces at hand, shows what listens and who reaches it, and keeps a Mac awake, the same sections on every machine. This repository also authors two doctrines every `@jterrazz` repository follows, shipped to agents as skills. This page lists every page of the manual, by what the reader came to do.

## Start

| Page                                        | Answers                                        |
| ------------------------------------------- | ---------------------------------------------- |
| [Getting started](start/getting-started.md) | From a fresh Mac to a named Mac, then its apps |

## Guides

| Page                         | Answers                                                                            |
| ---------------------------- | ---------------------------------------------------------------------------------- |
| [Develop](guides/develop.md) | Working from source: the gestures, which folder a change opens, what a change owes |
| [Test](guides/test.md)       | The three tiers, a use case told over a world, the adapter documents, the stubs    |
| [Release](guides/release.md) | The tagged release, how a machine takes it, provisioning from a checkout           |

## Concepts

| Page                                         | Answers                                                                                       |
| -------------------------------------------- | --------------------------------------------------------------------------------------------- |
| [Architecture](concepts/architecture.md)     | The families and the hexagon in each, the contracts, the Router, the engine, the fake world   |
| [Design system](concepts/design-system.md)   | The palette, the marks, the pipe format, Activity's graphs                                    |
| [Glossary](concepts/glossary.md)             | The one word for each concept, the words rejected, the words of the code                      |
| [The stack](concepts/stack.md)               | How `@jterrazz` projects compose: packages, naming, required files; shipped as `studio-stack` |
| [Repo structure](concepts/repo-structure.md) | Where knowledge lives in every `@jterrazz` repository; shipped as `studio-repo-structure`     |

## Reference

| Page                                              | Answers                                                                                       |
| ------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| [Commands](reference/commands.md)                 | Every command and flag, the hidden aliases, the plumbing, the exit codes                      |
| [Machines](reference/machines.md)                 | Macs, servers and workspaces, lifetimes, `--on` and `station use`, Tailscale, ssh, the engine |
| [Ports](reference/ports.md)                       | Private and Public, rights and audiences, reserved ports, sharing, HTTPS                      |
| [Accounts](reference/accounts.md)                 | Who is signed in, sharing an account with a VM, kept and bound secrets, Claude in a VM        |
| [Disk](reference/disk.md)                         | The Disk groups, the relocation rule, free, erase and move                                    |
| [Activity](reference/activity.md)                 | What a machine is doing, where each value comes from, the history and its graphs              |
| [Awake](reference/awake.md)                       | Every Mac's mode, keeping a Mac awake, the guards, Never sleeps                               |
| [The catalog's item format](reference/catalog.md) | Every key of an item file, the selector, the methods, options, an example per method          |
| [Machinefiles](reference/machinefiles.md)         | What a machine carries: every key, the overlay, sources and layers, the resolution, `explain` |
| [Settings](reference/settings.md)                 | Every setting station applies, the Checked group, how a setting asks its values               |
| [Preferences](reference/preferences.md)           | `preferences.json`, its groups and their owners, the environment variables                    |
| [Apps and skills](reference/apps-and-skills.md)   | The catalog, the method table, mise, skills as apps, the toolbelt                             |
| [Images](reference/images.md)                     | The VM images, the recipe, the binary inside                                                  |
| [Dotfiles](reference/dotfiles.md)                 | The embedded application configs and the `~/.zshrc` block                                     |
| [The API](reference/api.md)                       | `station.api/v2`: the transport, the frames, `calls.json`, the engine's own calls             |
| [The status contract](reference/status-json.md)   | `station.status/v1`, kept for one release behind the hidden `station status --json`           |
| [Footprint](reference/footprint.md)               | Everything `station` and the apps write on a machine, generated from the table                |

## Decisions

The records of decisions this repository alone took go in [`decisions/`](decisions/), numbered in the order they were taken, the mold `_template.md` beside them.

- [001 station mirrors the studio CLI's layers](decisions/001-station-mirrors-the-studio-cli-layers.md)
- [002 brew bundle installs, native commands uninstall](decisions/002-brew-bundle-installs-native-commands-uninstall.md)
- [003 mise owns the runtimes](decisions/003-mise-owns-the-runtimes.md)
- [004 station status --json is a contract](decisions/004-station-status-json-is-a-contract.md)
- [005 The version is stamped from the tag](decisions/005-the-version-is-stamped-from-the-tag.md)
- [006 Tart is the engine, behind a port](decisions/006-tart-is-the-engine-behind-a-port.md)
- [007 Files live in the XDG folders](decisions/007-files-live-in-the-xdg-folders.md)
- [008 Every VM has a lifetime](decisions/008-every-vm-has-a-lifetime.md)
- [009 One bootstrap verb, and no values in argv](decisions/009-one-bootstrap-verb-and-no-values-in-argv.md)
- [010 VMs are reached through their Mac](decisions/010-vms-are-reached-through-their-mac.md)
- [011 One engine per Mac](decisions/011-one-engine-per-mac.md)
- [012 Ports are reserved and shared by their Mac](decisions/012-ports-are-reserved-and-shared.md)
- [013 Every port leaves through Serve](decisions/013-every-port-leaves-through-serve.md)
- [014 The code is families, each a hexagon](decisions/014-families-and-contracts.md)
- [015 station's own choices are preferences.json](decisions/015-preferences-json.md)
- [016 Every machine is reached by its engine's API](decisions/016-api-v2-and-remote-by-api.md)
- [017 A VM is a server or a workspace](decisions/017-servers-and-workspaces.md)
- [018 A port is Private or Public](decisions/018-two-port-classes.md)
- [019 Activity is a section, its history the engine's memory](decisions/019-activity.md)
- [020 Use cases are proved over a fake world](decisions/020-the-fake-world.md)
- [021 The menu bar keeps its system window, its own focus walk, and a still status item](decisions/021-the-menu-bar-keeps-its-system-window.md)
- [022 HTTPS links are a Mac's choice, through Serve](decisions/022-https-links-are-a-macs-choice.md)
- [023 A VM's CPU on its Mac is the process that holds its guest's disk](decisions/023-a-vms-cpu-is-its-guests-process.md)
- [024 Agents run in Herdr, and tmux is no adapter](decisions/024-agents-run-in-herdr.md)
- [025 Agents have their own lifecycle, apart from their VM](decisions/025-agents-have-their-own-lifecycle.md)
- [026 The stories are the menu bar's test suite](decisions/026-the-stories-are-the-menu-bars-test-suite.md)
- [027 The agent hook keeps a scrubbed line, never code or commands](decisions/027-the-agent-hook-keeps-a-scrubbed-line.md)
- [028 A server's card shows its latency, not its traffic](decisions/028-a-servers-card-shows-latency-not-traffic.md)
- [029 An agent is answered and merged from the panel](decisions/029-an-agent-is-answered-and-merged-from-the-panel.md)
- [030 init makes a machine known; apply carries its Machinefile](decisions/030-init-makes-a-machine-known-apply-carries-its-machinefile.md) (Proposed)
- [031 The catalog defines, the Machinefile chooses](decisions/031-the-catalog-defines-the-machinefile-chooses.md) (Proposed)
- [032 The words of a VM's lifetime and network](decisions/032-the-words-of-a-vms-lifetime-and-network.md) (Proposed)
- [033 A machine is declared by its Machinefile](decisions/033-a-machine-is-declared-by-its-machinefile.md) (Proposed)
- [035 A port is a right with an audience](decisions/035-a-port-is-a-right-with-an-audience.md) (Proposed)
- [036 The menu bar owns its panel, and the panel's one background is a tinted glass](decisions/036-the-menu-bar-owns-its-glass-panel.md) (Proposed)
- [037 The TUI is retired; the CLI is the one terminal surface](decisions/037-the-tui-is-retired.md) (Proposed)

A decision spanning several repositories is recorded by the corpus that spans them. What was left for later is `TODO.md` at the repository root.
