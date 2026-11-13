# Agent brief: `station` (studio-station)

A Go CLI that sets up and runs a Mac, its VMs (servers and workspaces) and the owner's other Macs: apps, settings, disk, ports, accounts, background tasks, Awake, activity, images. This file **routes**; it does not restate what the corpus already says.

## Mental model

- **Families first.** `apps/engine/` is one package per family (`machines`, `apps`, `settings`, `ports`, `awake`...), each with its model, `contract/`, `calls/` and `adapter/<tech>/`; the architecture test holds the tree and every import ([`docs/concepts/architecture.md`](docs/concepts/architecture.md)).
- **A use case is a call.** `<family>.<verb>`, reached through the Router, which runs it in this process, at its target, or in a Mac's engine; every surface makes the same call.
- **Add to the catalog, not to command code.** An app (agent skills included, its disk row in the same file) or a setting is one package file, `catalog/packages/<category>/<name>.yaml` (`station.package/v1`); images are still values in `apps/engine/catalog/`, and a dev server's port and env are its package's `ports`.
- **Capabilities, never model names.** What a machine can do is its `Profile`; a section a kind does not have is refused as not for this machine.
- **The binary is hermetic.** The catalog's packages, the built-in Machinefiles, dotfiles and VM recipes are embedded; nothing reads a file inside the clone. The owner's packages and Machinefiles (`~/.config/station/`) and the shared source are layers above them ([`docs/reference/machinefiles.md`](docs/reference/machinefiles.md)).
- **The words are the glossary's.** One word per concept ([`docs/concepts/glossary.md`](docs/concepts/glossary.md)); the retired words are refused by `specs/architecture/`.
- **A use case is proved over a world.** Every outcome of every call has its story under `specs/usecases/`, told over the fake world; an adapter's argv is proved once in `specs/adapters/`.

## Where knowledge lives (route here first)

The corpus is `docs/` + `README.md`, mapped by `docs/README.md`. Do not duplicate it; link to it.

| Working on                                           | Read                                |
| ---------------------------------------------------- | ----------------------------------- |
| A fresh machine: install, init                       | `docs/start/getting-started.md`     |
| The families, contracts, Router, engine, world       | `docs/concepts/architecture.md`     |
| Palette, marks, pipe format, graphs                  | `docs/concepts/design-system.md`    |
| What a word means                                    | `docs/concepts/glossary.md`         |
| Building, and which folder a change opens            | `docs/guides/develop.md`            |
| The three tiers, use cases over a world, the stubs   | `docs/guides/test.md`               |
| The menu bar's stories, goldens, the Station Catalog | `docs/guides/test.md`               |
| The tagged release, `make provision`                 | `docs/guides/release.md`            |
| Every command and flag                               | `docs/reference/commands.md`        |
| Macs, servers, workspaces, `--on`, `use`, the engine | `docs/reference/machines.md`        |
| Ports: rights, audiences, sharing, HTTPS             | `docs/reference/ports.md`           |
| Accounts, sharing with a VM, secrets                 | `docs/reference/accounts.md`        |
| The Disk groups, free, erase, move                   | `docs/reference/disk.md`            |
| Activity, its history and graphs                     | `docs/reference/activity.md`        |
| Awake, the guards, Never sleeps                      | `docs/reference/awake.md`           |
| The settings and the Checked group                   | `docs/reference/settings.md`        |
| `preferences.json`, the environment variables        | `docs/reference/preferences.md`     |
| Apps, methods, skills, the toolbelt                  | `docs/reference/apps-and-skills.md` |
| Every key of a catalog package file                  | `docs/reference/catalog.md`         |
| Every key of a Machinefile, its layers, `explain`    | `docs/reference/machinefiles.md`    |
| VM images and their recipe                           | `docs/reference/images.md`          |
| The shell block, the embedded dotfiles               | `docs/reference/dotfiles.md`        |
| `station.api/v2`, `calls.json`, the engine's calls   | `docs/reference/api.md`             |
| The `station.status/v1` contract (one release)       | `docs/reference/status-json.md`     |
| What station writes on a machine                     | `docs/reference/footprint.md`       |
| Why a choice was made                                | `docs/decisions/`                   |
| What is left for later                               | `TODO.md`                           |
| `@jterrazz` stack conventions                        | `docs/concepts/stack.md`            |
| Repo doctrine (corpus/injection/compiler)            | `docs/concepts/repo-structure.md`   |

The repo-structure doctrine is authored **here** and ships to agents as the `studio-repo-structure` skill. This repo is an application: it adopts the corpus + injection layers and generates no API reference. Its one compiler is `make generate` (`apps/engine/cmd/generate/`), which projects the catalog into the `studio-toolbelt` skill, `docs/reference/footprint.md` and the JSON Schemas `packages/api/schemas/station.package.v1.json` and `packages/api/schemas/station.machinefile.v1.json`, writes the page fixtures of `apps/engine/ui/testdata/pages/` and the story worlds' pages of `apps/menubar/stories/pages/` from the real page functions, and the Registry into `apps/engine/remote/api/calls.json`.

## Setup & commands

```bash
make install    # Go modules + the npm half
make build      # Build .artifacts/go/station
make check      # lint (gofmt, golangci-lint, deadcode, typescript check), test, build-linux, then menubar-check (apps/menubar: swift test, the bundle)
make fix        # gofmt -w, then typescript fix
make generate   # the projections: the toolbelt skill, footprint, the catalog schema, page fixtures, calls.json
make provision  # put this build on the machine as the real station, then station init
```

`make install` never touches the machine; `make provision` does.

## Standing rules

- **Add to the catalog, not to command code.** Which folder a change opens is the table in `docs/guides/develop.md`.
- **Four things land in the same commit as the change that makes them true**: the `make generate` projection, the story that proves the behaviour, the page the behaviour falsified, and the skill a doctrine page feeds. Each is stated once, in `docs/guides/develop.md` § What a change owes.
- **An allowlist of `specs/architecture/` only shrinks.** A change that would add a line to one stops and asks.
- **Never edit a generated section by hand**: the blocks between the `GENERATED` markers of `skills/studio-toolbelt/SKILL.md` come from `make generate`, and `docs/reference/footprint.md`, the two `packages/api/schemas/*.json` and `apps/engine/remote/api/calls.json` are generated whole.
