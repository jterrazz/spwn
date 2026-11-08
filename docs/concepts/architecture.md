# Architecture

`station` is one Go binary whose tree names what it does: one top-level package per family (`machines`, `apps`, `settings`, `ports`, `awake`...), and inside each the same four parts, its model, its contracts, its calls and its adapters. A surface (the CLI, the TUI, the engine's server, the menu bar through it) turns a gesture into a call, the Router decides on which machine the call runs, and a pure page function turns its result into what is printed. The architecture test holds the tree and the direction of every import ([ADR-014](../decisions/014-families-and-contracts.md)).

| Section                    | Answers                                                           |
| -------------------------- | ----------------------------------------------------------------- |
| The tree                   | The packages under `src/`, families first                         |
| The hexagon in each family | Model, contracts, calls, adapters; strategies; what applies where |
| The contracts              | Every contract, the family that owns it, its adapters             |
| The Router                 | Where a call runs: this process, the target, a Mac's engine       |
| The engine                 | The process on each Mac, its jobs and its events                  |
| The fake world             | The spec build, and the world every fake reads                    |
| The path of a request      | What happens between argv and the bytes on the terminal           |
| Who may import whom        | The rules of `specs/architecture/`                                |
| The error model            | The one error shape, its types, the exit codes                    |
| The catalogue              | Why a feature is a value                                          |
| The state                  | The stores station keeps on a machine                             |
| The compiler               | What `make generate` projects                                     |

## The tree

The families, each with its model, `contract/`, `calls/` and `adapter/<tech>/`:

```text
src/
  machines/  apps/  settings/  disk/  ports/  accounts/  background/  awake/  activity/  images/
  agents/    the coding agents of a workspace, each named and in a terminal of its own: Judge folds
             their hook events into a state, Scrub keeps only what privacy lets through; contracts
             Hooks, Journal, Session, Forge, Repo
  overview/  the Overview's read (overview/calls), which composes every section's own read
  setup/     setup.run, the one bootstrap call, and the preferences groups it writes
  engine/    the engine's model (its events, its own calls), engine/calls, engine/contract (the Notifier)
    process/ the engine itself: its runs, its guest connections, its jobs, its socket and its server
    jobs/    the sample job, which feeds Activity's history
    feeds/   the menu bar's pages, built over watches
```

The shared kernel, which does no I/O:

```text
  core/
    machine/   a machine's Ref, Kind, System, Capabilities and Profile, and the Verdict of a rule
    section/   the sections and section.For(kind), the one table of which kind shows which
    finding/   one Needs you line, and finding.Rank, its one order
    confirm/   the graded confirmations: confirm.For(action, kind) and confirm.Require
    call/      a use case as a Call or a Watch, Bind and Gate, the Registry, the Router; lanes/, the read pool
    contract/  the contracts every family shares: Self, Preferences
    store/     what every store shares: its errors, its version, the Migrator setup runs
    fail/      the error: its type, its code, its words
    adapter/   macos and linux (Self), jsonfile (Preferences), fake
```

Between machines, and the machine's own door:

```text
  remote/    the Router; api/ (station.api/v2 frames, the client, calls.json); serve/ (their server);
             each/ (a call fanned out to many machines); contract/ and adapter/ (unix, sshstdio)
  platform/
    local/   Runner, FS, Env, Net, Sockets, Clock: the only importer of os/exec, files, the network;
             localtest/, its scripted fakes
    darwin/  the only cgo: IOKit (battery, lid, thermal, assertions) and the kernel tables, with their spec seams
```

The data, the presentation and the surfaces:

```text
  catalog/   pure data: apps and their disk rows, settings, managers, images, profiles, dotfiles, recipes
  page/      the page language: sections, rows, values, marks, reasons, actions; station.page/v1
  pages/     one package per page: a family's result to a page, pure
  ui/        token, atom, molecule, organism, template; render/ (tty, pipe, json, tui); help/
  output/    the only place that knows a terminal: picks the form, prints the one error block
  surface/
    cli/     the cobra tree, one folder per noun and verb; kit/; words/ (every code's words)
    tui/     the Bubble Tea app and its screens/
  world/     the fake world: data only, and its closed list of events
  wire/      the composition root: adapters, stores, the Profile, the Registry, the Router, Fang
  cmd/       station/ (process entry: argv, environment, streams, once), generate/ (the compiler)
```

Beside `src/`: `apps/menubar` (Swift, a client of the engine's socket), `specs/` (the proofs, [Test](../guides/test.md)), `skills/` (what this repository ships to agents).

## The hexagon in each family

Every family has the same four parts, named the same way, so a reader sees the business first and the plumbing second:

| Part              | Holds                                                                                                    | May import                                                       |
| ----------------- | -------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| the root          | the model and its rules: values, judgements (`settings.Judge`, `awake.Active(caps)`), no I/O             | `core/*`, `catalog`, other families' models                      |
| `contract/`       | one interface per need (`Hypervisor`, `Jobs`, `Meter`...), its preference group, and `contracttest/`     | the model, `core`                                                |
| `calls/`          | the use cases, each a `call.Call` or `call.Watch` named `<family>.<verb>`, and `All`, the family's list  | the model, its contracts, other families' `contract` and `calls` |
| `adapter/<tech>/` | one adapter per technology (`tart`, `launchd`, `brew`...); `fake/`, over the world; `jsonfile/`, a store | its contract, `platform/local`                                   |

A call declares its name, its section (empty for a call of no section: `machines.*`, `images.*`, `engine.*`, `setup.run`), its `Where`, its `Applies` (a `machine.Applies` rule over the target's Profile), the confirmation it may require, the error codes it may end in, and its `Run`. `wire` binds each family's `All` to its dependencies and hands them to the Registry. Before every call the gate runs (`call.Gate`): the target's kind must list the call's section in `core/section.For`, and the call's own `Applies` must say yes; a no is `not-for-this-machine`, exit 2, with the reason in words.

What a machine can do is its `core/machine.Profile`, built once by `wire`: its kind and system from `Self`, its battery, lid and thermal from `awake/contract.Power`, a job manager, a hypervisor and an engine from what it wires. A rule reads the capabilities, never a model name: a desktop Mac has no battery guard because it reports no battery, and a Linux workspace has no Background because it reports no job manager.

A strategy is a contract with several adapters chosen per item at run time, through one registry in the contract package: `Installers` of `src/apps/contract/` (per catalogue method), `settings/contract.Appliers` (per effect), `ports/contract.PublicRoutes` (per `--via` route). No other code switches on a method, an effect or a route. The `Hypervisor` is one contract with one real adapter, `tart`; the spec build runs the VM stories a second time over a fake `vz`, so a second hypervisor is a folder and a line in `wire`. A family may import another family's `contract` (Settings' `lock-after-login` writes through `background/contract.Jobs`, its `agent-hooks` through `agents/contract.Hooks`), never its adapters.

## The contracts

Each contract has a real adapter, a fake in its family's `adapter/fake/`, and a suite in its `contract/contracttest/` that both pass; the rule `contracts-are-whole` fails a contract missing any of the three.

| Family       | Contracts                                                                                                | Adapters                                                                                                                                             |
| ------------ | -------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `machines`   | `Hypervisor`, `Guest` and `GuestDialer`, `CheckRunner`, `SSHConfig`, `Unlocker`, `Waker`, `MachineStore` | `tart`; `ssh` (one x/crypto connection per VM), `dial`; `openssh` (the `~/.ssh/config` block); `macos`; `udp` (the Wake-on-LAN packet); `jsonfile`   |
| `images`     | `ImageSource`, `ImageStore`                                                                              | `oci` (ghcr); `jsonfile`                                                                                                                             |
| `apps`       | `Installer` (strategy), `Bundler`, `Lined`                                                               | `brew`, `mas`, `mise`, `npm`, `bun`, `uv`, `skills`, `script`, `shell`, `app` (presence only)                                                        |
| `settings`   | `Applier` (strategy), `Inspector`, `AppliedStore`                                                        | `dotfile`, `touch`, `gitaliases`, `zshblock`, `defaults`, `vault`, `neversleeps`, `agenthooks`, `docker`, `git`, `macos`; `cli`, `linux`; `jsonfile` |
| `disk`       | `DiskUsage`                                                                                              | `apfs` (macOS), `posix` (Linux), `docker`                                                                                                            |
| `ports`      | `Listeners`, `Tailnet`, `PublicRoute` (strategy), `ReservationStore`, `DeclarationStore`                 | `lsof` (macOS), `procnet` (Linux), `docker`, `guest`; `tailscale`; `cloudflare`, `funnel`, `lan`; `jsonfile`                                         |
| `accounts`   | `Identities`, `Vault`, `SecretSource`, `ShareStore`                                                      | `cli` (catalogue-driven sign-in reads), `tailscale`; `keychain`; `studio`; `jsonfile`                                                                |
| `background` | `Jobs`                                                                                                   | `launchd`                                                                                                                                            |
| `awake`      | `Power`                                                                                                  | `macos` (IOKit through `platform/darwin`, `pmset`)                                                                                                   |
| `activity`   | `Meter`                                                                                                  | `macos` (sysctl, IOKit block devices, proc_pidinfo), `linux` (`/proc`)                                                                               |
| `agents`     | `Hooks` (strategy), `Journal`, `Session`, `Multiplexer`, `Forge`, `Repo`                                 | `claudecode`, `codex`; `jsonfile`; `herdr`; `gh`; `git`                                                                                              |
| `engine`     | `Notifier`                                                                                               | `log` (the engine log)                                                                                                                               |
| `remote`     | `Endpoint` and `EndpointDialer`                                                                          | `unix` (this Mac's engine socket), `sshstdio` (another Mac: `ssh <mac> station engine attach`)                                                       |
| `core`       | `Self`, `Preferences`                                                                                    | `macos`, `linux`; `jsonfile` (`preferences.json`)                                                                                                    |

Not a contract, on purpose: the activity history and the engine's run registry live in `engine/process` and are reached through the small consumer-side interfaces a family's `calls` declares (`machines/calls.Runs`, `activity/calls.History`); there is nothing to swap and nothing to fake. Infrastructure is not a contract either: `Runner`, `FS`, `Env`, `Net`, `Sockets` and `Clock` are `platform/local`'s, importable by adapters only, so no call and no model can run a command or open a file.

## The Router

Every surface reaches a call through `core/call.Router` (`Invoke`, `Watch`); `call.Invoke[D, In, Out]` keeps the call's dependencies out of the surface's sight. `remote.Router`, which `wire` builds from the Registry and the two `EndpointDialer`s, resolves a call by its `Where` and its target:

| Target                         | `Where`              | Runs                                                                                                  |
| ------------------------------ | -------------------- | ----------------------------------------------------------------------------------------------------- |
| any                            | `in-process`         | in this process; the call itself may fan out to many machines through `remote/each`                   |
| this machine                   | `at-target`          | in this process                                                                                       |
| a VM of this Mac               | `at-target`          | this Mac's engine, over its socket, which relays the frame to the VM's guest over its one connection  |
| this Mac, or a VM of this Mac  | `in-engine`          | this Mac's engine, over its unix socket                                                               |
| another Mac, or one of its VMs | any but `in-process` | that Mac's engine, dialed by `ssh <mac> station engine attach`, which resolves the frame the same way |

So there is one way into a VM (its Mac's engine and its one `Guest` connection) and one way to another Mac (`sshstdio`); station never runs `ssh <vm>` ([ADR-016](../decisions/016-api-v2-and-remote-by-api.md)). The far side returns data, and this binary renders it with its own pages, so `station apps --on linux-1` looks exactly like `station apps` on linux-1. Each connection opens with `Hello`: the far version and the calls it serves; a call it does not serve is `far-too-old`, with the command that updates it. A confirmation travels in the call's input and the far side judges it with the same `confirm.For`. When this Mac's engine is down an action that needs it is refused with `engine-stopped`; a read never is, it returns what it can read and the `Engine stopped` finding, exit 0. Inside a VM, station targets the VM itself, and its few `in-engine` calls (`ports.share`) reach its Mac's engine through the socket the engine offers inside the VM over the same guest connection, so a VM holds no key to its Mac. All machines is `remote/each.Do`: 4 machines at a time, 5 s each, an unreachable Mac a finding of its own. The frames are [The API](../reference/api.md)'s.

## The engine

`engine/` is the process launchd keeps alive on each Mac (`station engine run`, the agent `com.jterrazz.station.engine`). It is a family like the others, plus a process: `engine` holds its model (its `Event`s, the names and shapes of its own calls), `engine/calls` its calls (`engine.read`, `engine.start`, `engine.stop`), `engine/contract` the `Notifier`. `engine/process` is the engine itself: it holds the runs of every VM, the one guest connection of each running VM, the Awake session, the activity history, the reservations' listeners and the event bus, serves `remote/serve` on its socket, and runs its jobs. The jobs hold no product rule of their own: each calls its family's rules and calls.

| Job        | Runs                                                   | Does                                                                                                                |
| ---------- | ------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------- |
| `keep`     | when a run ends, at start, on the engine's loop        | keeps each VM in its wanted state: a server boots again after its wait; the health steps after every boot           |
| `guest`    | per running VM                                         | holds its one `Guest` connection: dial after boot, redial, close at the end; every other job uses it                |
| `share`    | on a reservation change, on the loop                   | reconciles what Tailscale Serve shares with the reserved ports                                                      |
| `expire`   | every minute                                           | takes back a shared account whose term ended                                                                        |
| `awake`    | on a reading change, on `never_sleeps`, on the loop    | holds the keep-awake session and Never sleeps (`Power.Hold`), ends them by the guards this Mac's capabilities allow |
| `sample`   | every 2 s while a surface watches a machine, else 60 s | feeds Activity's history (`engine/jobs`)                                                                            |
| `exposure` | every 5 min                                            | reads every listener of this Mac and its VMs; a port reachable outside Tailscale undeclared is told                 |

An event (`engine.Event`: its name, its machine, its text, its severity) is the one channel the `Notifier` and the menu bar read: `server_restarting`, `server_loop`, `server_powered_off`, `server_back`, `awake_ended`, `port_exposed`, `disk_low`. The menu bar's pages are built in `engine/feeds` over watches, and streamed to it as `station.page/v1`.

## The fake world

`src/world` is the fake world: one serialisable value holding, for each Mac and VM, one slice per family, the image registry, the clock, and the faults scripted for the fakes; its events are a closed list (`reboot`, `power-cut`, `crash`, `listen`, `advance`, `unplug`...). The spec build (`-tags world`, never shipped) is the one build that wires it: under `$STATION_WORLD`, `wire/world_on.go` hands every family's `adapter/fake` over its machine's slice where the release build's `wire/world_off.go` keeps the real adapter, and steps each Mac's engine over the world's clock. How a use case is told over it is [Test](../guides/test.md#a-use-case-over-a-world)'s, and why is [ADR-020](../decisions/020-the-fake-world.md).

## The path of a request

`station apps install node`:

1. `cmd/station` hands argv, the environment and the streams to `wire.Run`.
2. `wire` builds this machine's Profile, the stores, every family's calls over the real adapters, the Registry and the Router, then the cobra tree, and runs it under Fang.
3. The leaf `surface/cli/apps/install` binds argv to the call's input; argv's own refusals exit 2 before any question, any read and any effect.
4. The leaf invokes `apps.install` through the Router. The target is this machine (`--on` and `station use` change it), so the call runs in this process, past the gate. The call orders the dependencies, runs each method's installer from the `Installers` registry, and reads the inventory back.
5. A call that confirms decides it in its own `Run` (`confirm.Require`); when it refuses with `needs-confirmation`, kit answers: `--yes`, the leaf's question on a terminal, or a refusal, then invokes it once more with the answer. No surface decides whether to ask.
6. The leaf hands the result to its page function (`pages/apps`), which returns a `page.Page`, and to its JSON document for `--json`.
7. `output` prints it in the form the stream allows: tty with colour, pipe as tab-separated records, json (`station.<call>/v1`), or the page itself as `station.page/v1` under the hidden `--page` (what the TUI renders). `NO_COLOR` degrades a terminal to the pipe form.

Multi-step behaviour (install then configure, restart then wait) lives in the call, so a CLI leaf, a TUI key and a menu-bar click make the same one call.

## Who may import whom

The rules are Go tests under [`specs/architecture/`](../../specs/architecture/); each proves it still bites by judging one violating file in memory, and each rule that lists today's offenders keeps them in an allowlist that only shrinks: a line no longer true fails the gate until it is removed. This page names them and restates none.

| Rule                                                                     | File                                                   | Holds                                                                                                                                                                   |
| ------------------------------------------------------------------------ | ------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `tree-screams`                                                           | `tree_test.go`                                         | the top of `src/` is families and the named shared packages; inside a family, only its four parts; every call sets `Applies`                                            |
| `imports`                                                                | `imports_test.go`                                      | the families' import table: a model reaches no platform, an adapter only its own contract, a surface the engine only through `engine/calls`                             |
| the layers                                                               | `layers_test.go`                                       | the presentation's table (page, pages, ui, output, surfaces, wire) and who may hold a symbol: `os/exec`, files, the clock, a colour, Fang, cgo                          |
| `technology-words`, `no-model-names`                                     | `words_test.go`, `models_test.go`                      | a technology is named only in its adapter; no rule reads a Mac's model name                                                                                             |
| `contracts-are-whole`                                                    | `contracts_test.go`                                    | every contract has a real adapter, a fake and a suite both pass                                                                                                         |
| `calls-gated`, `confirm-in-usecase`, `every-code-worded`                 | `calls_test.go`                                        | every call is in its family's `All` once, named `<family>.<verb>`; a confirmation is required by its call alone; every `fail.Code` has its words in `surface/cli/words` |
| `one-way-into-a-vm`                                                      | `vm_test.go`                                           | a VM is reached by its engine's one `Guest` connection: no `scp`, no `ssh` exec but to a Mac                                                                            |
| `one-writer-per-store`                                                   | `stores_test.go`                                       | only `wire` builds a store's `jsonfile`, and a store or a preference group is written only by its family's calls                                                        |
| `no-outbound-literals`                                                   | `outbound_test.go`                                     | a machine's read calls no third-party server                                                                                                                            |
| `no-remote-reexec`, `engine-no-tcp`                                      | `remote_test.go`, `tcp_test.go`                        | no command is executed again on another machine to read its text; the engine listens on no TCP port                                                                     |
| `no-sudo-literal`, `no-subprocess-in-sample`, `fixtures-carry-no-secret` | `sudo_test.go`, `sample_test.go`, `fixtures_test.go`   | station never runs sudo (a fix names the command for you); the sample job runs no program; no fixture holds a token                                                     |
| `tui-loading-law`                                                        | `tui_test.go`                                          | the TUI's reads run only inside a command, never in a view                                                                                                              |
| `coverage`                                                               | `coverage_test.go`                                     | every call tells a story of each outcome it declares under `specs/usecases/`                                                                                            |
| `spec-build-never-ships`                                                 | `release_test.go`                                      | the release binary links no world and no fake; the spec build links every family's fake                                                                                 |
| the words                                                                | `banned_test.go`, `vocabulary_test.go`, `help_test.go` | the retired words never come back in a page, a help line or an error word                                                                                               |
| the calls' list                                                          | `api_test.go`                                          | `calls.json` is what the Registry holds                                                                                                                                 |

## The error model

Every failure is one `core/fail` error: its type, its code (`confirm.mismatch`, `account.not-signed-in`), its subject, and its words, kept in `surface/cli/words` (one per code) and `surface/cli/kit` (`failures_<family>.go`, for the errors a family's adapters raise). It prints as one block on stderr, and nothing else writes an error:

```text
Error: no app named "nope"
Hint: station apps
```

With `--json` the same error is also the `station.error/v1` document on stdout. The exit code comes from the type, never from a call to exit inside a command:

| Code | Types                                                                                  |
| ---- | -------------------------------------------------------------------------------------- |
| 0    | done, or a read that found findings (an unreachable machine is a finding)              |
| 1    | `failed`, `unreachable`, `engine-stopped`, `far-too-old`, `declined`                   |
| 2    | `refused`, `not-found`, `not-for-this-machine`, `needs-confirmation`, `needs-terminal` |
| 130  | `interrupted`; no block is printed                                                     |

Fang renders the help, the version and the completion script, and handles the interrupt, behind the CLI's own error layer: `wire` is its only importer, so removing it is one call. `station --version` prints `station <version> (<commit>)`, stamped by the linker from the tag ([ADR-005](../decisions/005-the-version-is-stamped-from-the-tag.md)).

## The catalogue

The catalogue is values in `src/catalog/`: apps (agent skills included) and each app's disk row, settings, managers, images, ports' profiles. A value carries no function, and a test says so: behaviour chosen by data belongs to a family, picked by the value's type from a closed union. **Add to the catalogue, not to command code**: one app entry gives the app a row in `station apps`, a count in the Overview, an install, a remove, an update and its Disk row, because all of them read the same value. Every entry says where it lives on (`mac-only`, `every-machine`, `vms-only`), so one binary lists and installs only what fits its machine. Which folder a change opens is [Develop](../guides/develop.md#where-a-change-goes)'s.

## The state

The binary holds the catalogue; the machine holds its own state, in the XDG folders, each store one contract of its family with one `jsonfile` adapter, built by `wire` alone:

| Store              | Family     | File                                                                              |
| ------------------ | ---------- | --------------------------------------------------------------------------------- |
| `Preferences`      | `core`     | `~/.config/station/preferences.json` ([Preferences](../reference/preferences.md)) |
| `DeclarationStore` | `ports`    | `~/.config/station/public.json`                                                   |
| `MachineStore`     | `machines` | `~/.local/state/station/vms/`                                                     |
| `ImageStore`       | `images`   | `~/.local/state/station/images/`                                                  |
| `ShareStore`       | `accounts` | `~/.local/state/station/shares.json`                                              |
| `ReservationStore` | `ports`    | `~/.local/state/station/leases.json`                                              |
| `AppliedStore`     | `settings` | `~/.local/state/station/applied/`                                                 |
| `UptimeStore`      | `machines` | `~/.local/state/station/uptime.json`, written by the engine's keep job alone      |
| `Journal`          | `agents`   | `~/.local/state/station/agents/`, inside a VM, one file per session               |

Each store carries its version; `station setup` runs every store's `Migrator`, and a store written by a newer station is refused rather than read. Nothing else is consulted, and no file inside the clone matters: the dotfiles and the VM recipes are embedded. [Footprint](../reference/footprint.md) lists every file station writes.

## The compiler

This repository is an application in the sense of the [repo-structure doctrine](repo-structure.md): it authors a corpus and generates no API reference. Its compiler is `make generate` (`src/cmd/generate/`), which writes, from the catalogue and the Registry: the rosters between the `GENERATED` markers of `skills/studio-toolbelt/SKILL.md`, `docs/reference/footprint.md` whole, the page fixtures of `src/ui/testdata/pages/` from the real page functions, the pages of the menu bar's story worlds (`src/cmd/generate/pages/stories.go`: the `machines` world of every Mac and VM and its variants, the clock pinned at 2026-11-07 10:30) into `apps/menubar/stories/pages/<world>.<page>.json`, the menu bar's recorded streams, and `src/remote/api/calls.json`. Each projection is committed, and a test fails when one drifts.

## Related

- [Design system](design-system.md): what the pages look like in each form.
- [Develop](../guides/develop.md): which folder a change opens.
- [Test](../guides/test.md): the suites that prove the tree and the behaviour.
- [The API](../reference/api.md): `station.api/v2` and `calls.json`.
- [ADR-014](../decisions/014-families-and-contracts.md): why the tree is families.
