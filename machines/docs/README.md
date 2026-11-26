# The manual of Machines

Machines is the Spwn app that sets up and runs a Mac, its VMs and your other Macs, each from one file; its command is `station`. It installs apps and agent skills from a catalogue, applies and checks the machine's settings, keeps servers alive and workspaces at hand, shows what listens and who reaches it, and keeps a Mac awake, the same sections on every machine. What Spwn is as a whole is stated once, in the README at the root of the repository. This folder also authors two doctrines every `@jterrazz` repository follows, shipped to agents as skills. This page lists every page of the manual, by what the reader came to do.

## Start

| Page                                        | Answers                                        |
| ------------------------------------------- | ---------------------------------------------- |
| [Getting started](start/getting-started.md) | From a fresh Mac to a named Mac, then its apps |

## Guides

| Page                         | Answers                                                                            |
| ---------------------------- | ---------------------------------------------------------------------------------- |
| [Develop](guides/develop.md) | Working from source: the gestures, which folder a change opens, what a change owes |
| [Test](guides/test.md)       | The three tiers, a use case told over a world, the adapter documents, the stubs    |
| [Release](guides/release.md) | What a release carries, how a machine takes it, provisioning from a checkout       |

## Concepts

| Page                                         | Answers                                                                                       |
| -------------------------------------------- | --------------------------------------------------------------------------------------------- |
| [Architecture](concepts/architecture.md)     | The families and the hexagon in each, the contracts, the Router, the engine, the fake world   |
| [Design system](concepts/design-system.md)   | The palette, the marks, the pipe format, Activity's graphs                                    |
| [Glossary](concepts/glossary.md)             | The one word for each concept, the words rejected, the words of the code                      |
| [The stack](concepts/stack.md)               | How `@jterrazz` projects compose: packages, naming, required files; shipped as `studio-stack` |
| [Repo structure](concepts/repo-structure.md) | Where knowledge lives in every `@jterrazz` repository; shipped as `studio-repo-structure`     |

## Reference

| Page                                                 | Answers                                                                                        |
| ---------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| [Commands](reference/commands.md)                    | Every command and flag, the hidden aliases, the plumbing, the exit codes                       |
| [Machines](reference/machines.md)                    | Macs, servers and workspaces, lifetimes, `--on` and `station use`, Tailscale, ssh, the engine  |
| [Ports](reference/ports.md)                          | Private and Public, rights and audiences, reserved ports, sharing, HTTPS                       |
| [Accounts](reference/accounts.md)                    | Who is signed in, the vault, a grant to a VM, Claude in a VM                                   |
| [Disk](reference/disk.md)                            | The Disk groups, the relocation rule, free, erase and move                                     |
| [Activity](reference/activity.md)                    | What a machine is doing, where each value comes from, the history and its graphs               |
| [Journal](reference/journal.md)                      | What was done on each Mac and its VMs, kept thirty days, read by day, paged, narrowed          |
| [Awake](reference/awake.md)                          | Every Mac's mode, keeping a Mac awake, the guards, Never sleeps                                |
| [The catalog's package format](reference/catalog.md) | Every key of a package file, the selector, the methods, options, an example per method         |
| [Machinefiles](reference/machinefiles.md)            | What a machine carries: every key, the overlay, sources and origins, the resolution, `explain` |
| [Settings](reference/settings.md)                    | Every setting Machines applies, the Checked group, how a setting asks its values               |
| [Preferences](reference/preferences.md)              | `preferences.json`, its groups and their owners, the environment variables                     |
| [Apps and skills](reference/apps-and-skills.md)      | The catalog, the method table, mise, skills as apps, the toolbelt                              |
| [Images](reference/images.md)                        | The VM images, the recipe, the binary inside                                                   |
| [Dotfiles](reference/dotfiles.md)                    | The embedded application configs and the `~/.zshrc` block                                      |
| [The API](reference/api.md)                          | `station.api/v2`: the transport, the frames, `calls.json`, the engine's own calls              |
| [Footprint](reference/footprint.md)                  | Everything `station` and the apps write on a machine, generated from the table                 |

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
- [024 Agents run in Herdr, and tmux is no adapter](decisions/024-agents-run-in-herdr.md) (Superseded by 052)
- [025 Agents have their own lifecycle, apart from their VM](decisions/025-agents-have-their-own-lifecycle.md) (Superseded by 052)
- [026 The stories are the menu bar's test suite](decisions/026-the-stories-are-the-menu-bars-test-suite.md)
- [027 The agent hook keeps a scrubbed line, never code or commands](decisions/027-the-agent-hook-keeps-a-scrubbed-line.md) (Superseded by 052)
- [028 A server's card shows its latency, not its traffic](decisions/028-a-servers-card-shows-latency-not-traffic.md)
- [029 An agent is answered and merged from the panel](decisions/029-an-agent-is-answered-and-merged-from-the-panel.md) (Superseded by 052)
- [030 init makes a machine known; apply carries its Machinefile](decisions/030-init-makes-a-machine-known-apply-carries-its-machinefile.md) (Accepted)
- [031 The catalog defines, the Machinefile chooses](decisions/031-the-catalog-defines-the-machinefile-chooses.md) (Accepted)
- [032 The words of a VM's lifetime and network](decisions/032-the-words-of-a-vms-lifetime-and-network.md) (Accepted)
- [033 A machine is declared by its Machinefile](decisions/033-a-machine-is-declared-by-its-machinefile.md) (Accepted)
- [034 Apply carries a Machinefile; the ledger bounds removal](decisions/034-apply-carries-a-machinefile-the-ledger-bounds-removal.md) (Accepted)
- [035 A port is a right with an audience](decisions/035-a-port-is-a-right-with-an-audience.md) (Accepted)
- [036 The menu bar owns its panel, and the panel's one background is a tinted glass](decisions/036-the-menu-bar-owns-its-glass-panel.md) (Accepted)
- [037 The TUI is retired; the CLI is the one terminal surface](decisions/037-the-tui-is-retired.md) (Accepted)
- [038 An app is carried by one method, the one its plan shows](decisions/038-an-app-is-carried-by-one-method-the-one-its-plan-shows.md) (Accepted)
- [039 A share is shown under its audience](decisions/039-a-share-is-shown-under-its-audience.md) (Accepted)
- [040 apt takes back only what it names; a release installs only what a checksum vouches for](decisions/040-apt-takes-back-only-what-it-names-a-release-installs-only-what-a-checksum-vouches-for.md) (Accepted)
- [041 The lock is what apply installs; upgrade alone moves it](decisions/041-the-lock-is-what-apply-installs.md) (Accepted)
- [042 Versions are semver ranges, applied every night](decisions/042-versions-are-semver-ranges-applied-nightly.md) (Proposed)
- [043 A shell block is judged against what station last wrote](decisions/043-a-shell-block-is-judged-against-what-station-last-wrote.md) (Proposed)
- [044 An image is baked from its Machinefile; the recipe is plumbing alone](decisions/044-an-image-is-baked-from-its-machinefile.md) (Proposed)
- [045 The agents family retires, and an agent is a package](decisions/045-the-agents-family-retires-an-agent-is-a-package.md) (Superseded by 052)
- [046 The repository is a monorepo of members](decisions/046-the-repository-is-a-monorepo-of-members.md) (Proposed)
- [047 A catalog entry is a package, and a machine file is `<name>.machine.yaml`](decisions/047-a-catalog-entry-is-a-package-a-machine-file-is-name-machine-yaml.md) (Accepted)
- [048 The engine withholds a secret before any answer](decisions/048-the-engine-withholds-a-secret-before-any-answer.md) (Proposed)
- [049 A VM signs itself in, and the vault grants the rest](decisions/049-a-vm-signs-itself-in-the-vault-grants-the-rest.md) (Proposed)
- [050 The desktop app is laid out as a hexagon](decisions/050-the-desktop-app-is-laid-out-as-a-hexagon.md) (Accepted)
- [051 A machine is a base and its Machinefiles, one base per distro](decisions/051-a-machine-is-a-base-and-its-machinefiles.md) (Accepted)
- [052 Station keeps no agents; another binary owns them](decisions/052-station-keeps-no-agents.md) (Superseded by 088)
- [053 The panel is the untinted system glass over a hue-free grey, with no white card](decisions/053-the-panel-is-untinted-glass-over-a-grey-ground.md) (Accepted)
- [054 A machine's standing is one rule at each level, and every verdict reads it](decisions/054-a-machines-standing-is-one-rule-at-each-level.md) (Accepted)
- [055 Station reads only its current state, and retires no older one](decisions/055-station-reads-only-its-current-state.md) (Proposed)
- [056 Station names the Local Network approval a Mac withholds](decisions/056-station-names-the-local-network-approval-a-mac-withholds.md) (Proposed)
- [057 Init moves an app's whole old home, and says so and asks before it moves](decisions/057-init-moves-whole-homes-and-asks-first.md) (Proposed)
- [058 The window installs and starts the core itself](decisions/058-the-window-installs-and-starts-the-core-itself.md) (Accepted)
- [059 The latest release is read by two calls: one reads what the engine kept, one asks GitHub](decisions/059-the-latest-release-is-read-by-two-calls.md) (Accepted)
- [060 The window offers the engine's release and updates on a press](decisions/060-the-window-offers-the-engines-release-and-updates-on-a-press.md) (Accepted)
- [061 The history is kept by the Mac that holds the machine](decisions/061-the-history-is-kept-by-the-mac-that-holds-the-machine.md) (Proposed)
- [062 A failed create stays on the record until its start or its delete](decisions/062-a-failed-create-stays-on-the-record-until-its-start-or-its-delete.md) (Proposed)
- [063 A gesture is kept by the Router of its Mac, with who did it](decisions/063-a-gesture-is-kept-by-the-router-of-its-mac.md) (Proposed)
- [064 A key never leaves its device; station is the ssh agent](decisions/064-a-key-never-leaves-its-device.md) (Proposed)
- [065 The window updates itself through the Tauri updater](decisions/065-the-window-updates-itself-through-the-tauri-updater.md) (Superseded by 076)
- [066 A commit owes that it builds and lints, a lot owes the gate](decisions/066-a-commit-owes-that-it-builds-and-lints-a-lot-owes-the-gate.md) (Accepted)
- [067 One train, appended by makers, proved at its tip](decisions/067-one-train-appended-by-makers-proved-at-its-tip.md) (Accepted)
- [068 In the fake world no result waits on real time](decisions/068-in-the-fake-world-no-result-waits-on-real-time.md) (Proposed)
- [069 The menu bar hosts its panel once, and keeps what it saw](decisions/069-the-menu-bar-hosts-its-panel-once-and-keeps-what-it-saw.md) (Accepted)
- [070 The engine keeps one read model; pages and views are its projections](decisions/070-the-engine-keeps-one-read-model.md) (Accepted)
- [071 The window mirrors engine-kept state, never blank](decisions/071-the-window-mirrors-engine-kept-state-never-blank.md) (Accepted)
- [072 Before 1.0, every station runs the same release, and every Mac pushes its state](decisions/072-before-1-0-every-station-runs-the-same-release.md) (Accepted)
- [073 Spwn.app carries the engine on a Mac with a screen; install.sh is the headless install](decisions/073-spwn-app-carries-the-engine-on-a-mac-with-a-screen.md) (Accepted)
- [074 A far station answers its hello within 5 seconds, and ssh keeps the link alive](decisions/074-a-far-station-answers-its-hello-within-5-seconds.md) (Proposed)
- [075 A move merges into a cache and refuses onto anything else](decisions/075-a-move-merges-a-cache-and-refuses-onto-data.md) (Proposed)
- [076 Spwn on a machine: one active app, one background job, the engine updates](decisions/076-spwn-on-a-machine-one-active-app-one-background-job-the-engine-updates.md) (Accepted)
- [077 A workspace trusts its Mac's device key first](decisions/077-a-workspace-trusts-its-macs-device-key-first.md) (Proposed)
- [078 A server is set up from its Machinefile, on the owner's verb alone](decisions/078-a-server-is-set-up-from-its-machinefile-on-the-owners-verb.md) (Accepted)
- [079 A server runs the image phase, then a first-start phase](decisions/079-a-server-runs-the-image-phase-then-a-first-start-phase.md) (Accepted)
- [080 `kind: server` is a usable Machinefile](decisions/080-kind-server-is-a-usable-machinefile.md) (Accepted)
- [081 A server takes one grant, the Tailscale auth key, delivered once](decisions/081-a-server-takes-one-grant-the-tailscale-auth-key-delivered-once.md) (Accepted)
- [082 A server keeps its data on a disk of its own](decisions/082-a-server-keeps-its-data-on-a-disk-of-its-own.md) (Accepted)
- [083 A re-ask is quiet only for an engine-kept view, and never by a timer](decisions/083-a-re-ask-is-quiet-only-for-an-engine-kept-view-never-by-a-timer.md) (Accepted)
- [084 `base-macos` is fetched from Cirrus Labs, pinned by digest](decisions/084-base-macos-is-fetched-from-cirrus-labs-pinned-by-digest.md) (Accepted)
- [085 A Mac run alone keeps a device key that signs with nobody present](decisions/085-a-mac-run-alone-keeps-a-device-key-that-signs-with-nobody-present.md) (Accepted)
- [086 A VM reserves a package's port only when its Machinefile installs it](decisions/086-a-vm-reserves-a-packages-port-only-when-its-machinefile-installs-it.md) (Accepted)
- [087 A fetched macOS guest takes its key at its first login](decisions/087-a-fetched-macos-guest-takes-its-key-at-its-first-login.md) (Accepted)
- [088 Agents live in the one binary, as families reached by calls](decisions/088-agents-live-in-the-one-binary-as-families-reached-by-calls.md) (Accepted)
- [089 An agent runs in a Herdr terminal on its own machine](decisions/089-an-agent-runs-in-a-herdr-terminal-on-its-own-machine.md) (Accepted)
- [090 A machine is reached through its engine's door, never through OpenSSH](decisions/090-a-machine-is-reached-through-its-engines-door-never-through-openssh.md) (Accepted)
- [091 An adopted app is a state, never work](decisions/091-an-adopted-app-is-a-state-never-work.md) (Proposed)
- [092 A machine's device book says who may reach it](decisions/092-a-machines-device-book-says-who-may-reach-it.md) (Proposed)
- [093 The window is a path bar over a cool dark](decisions/093-the-window-is-a-path-bar-over-a-cool-dark.md) (Proposed)
- [094 Machines' Turborepo leaves Go to the Makefile](decisions/094-machines-turborepo-leaves-go-to-the-makefile.md) (Proposed)
- [096 Spwn is one app: its menu bar, its notifications and its engine live while it runs](decisions/096-spwn-is-one-app-its-menu-bar-notifications-and-engine-live-while-it-runs.md) (Proposed)
- [097 A confirmation follows its consequence](decisions/097-a-confirmation-follows-its-consequence.md) (Proposed)
- [098 A long act is guided by the engine; a Guide is a projection of what its call reports](decisions/098-a-long-act-is-guided-by-the-engine-a-guide-is-a-projection-of-what-its-call-reports.md) (Proposed)
- [104 A VM's disk sync follows what is lost](decisions/104-a-vms-disk-sync-follows-what-is-lost.md) (Proposed)

A decision spanning several repositories is recorded by the corpus that spans them. What was left for later is `TODO.md` at the repository root.
