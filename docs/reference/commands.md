# Commands

Every command of `station`, grouped as `station help` groups them. The words are the [Glossary](../concepts/glossary.md)'s.

| Section                   | Answers                                                                                        |
| ------------------------- | ---------------------------------------------------------------------------------------------- |
| Rules                     | What holds for every command                                                                   |
| Machines                  | `machines`                                                                                     |
| On a machine              | `overview`, `activity`, `apps`, `settings`, `disk`, `ports`, `accounts`, `background`, `awake` |
| A Mac's images and engine | `images`, `engine`                                                                             |
| This terminal             | `use`                                                                                          |
| Station itself            | `init`, `--version`, `help`                                                                    |
| Hidden aliases            | The v2 words that still run, for one release                                                   |
| Plumbing                  | The hidden commands the machine runs for itself                                                |
| Exit codes                | What 0, 1, 2 and 130 mean                                                                      |

## Rules

- Bare `station` opens the TUI on a terminal; piped, or with `--all` or `--json`, it prints `station overview`, exit 0.
- A bare noun lists (`apps`, `settings`, `disk`, `machines`, `ports`, `images`, `accounts`). A verb whose argument is missing is refused, its hint the command with it: `station apps install` hints `station apps install <app>`.
- An existing thing is a positional (`machines stop api`); a new thing is described by options (`machines create api --persistent`).
- One verb per meaning, the [Glossary](../concepts/glossary.md#the-verbs)'s: `apps` install, update, remove; `settings` apply, diff, off; `disk` free, erase, move; a VM create, start, stop, restart, edit, delete. The only other words are the [hidden aliases](#hidden-aliases).
- Every read takes `--json`. Every destructive action asks on a terminal or takes `--yes`.
- Every refusal exits 2 before any question; every failure is one `Error:` and one `Hint:`. With `--json`, the same refusal or failure is also the `station.error/v1` document on stdout (`schema`, `type`, `code`, `subject`, `message`, `hint`, `fix`), the exit code unchanged. A question answered no is `Error: <what> kept: you said no`, exit 1.
- A command that needs an app this Mac lacks is refused before anything runs, exit 2, naming the app and what needs it: `Error: tart is required by machines`, `Hint: station apps install tart`. The needs are the [Glossary](../concepts/glossary.md)'s: the VM verbs that make or run a VM and the image verbs need tart, `ports share` needs Tailscale, `accounts share` the studio CLI. On a terminal the refusal is one line, `○ tart is required by machines · station apps install tart`, then `Install tart now?`: yes installs it as `apps install` does and runs the command again; no ends there, exit 2. A need station installs none of (the studio CLI) is the refusal on a terminal too.
- A word station does not know is `Error: unknown command "<word>"` / `Hint: station help`, exit 2; so is `station help <word>`.
- A word under `disk` that names one of its entries, the verb forgotten, is still refused, but its hint is the command meant: `station disk go` is `"go" is a disk row: say what to do with it`, hint `station disk free go`. Under `apps` the entry is a read: `station apps jq` is jq's detail; under `settings`, a setting's name is its read: `station settings dns`.
- `--on <machine>` runs any command below on another machine, through its engine, and renders the answer on this one; see [Machines](machines.md#on-another-machine).
- `station use <machine>` makes every later command of this terminal run there, as `--on` would; `--on` still wins for one command. See [Machines](machines.md#this-terminals-machine).
- A value a command needs from you is never an argv flag when it could be a secret: a setting asks in a form on a terminal ([Settings](settings.md)).

## Machines

### `station machines`

```sh
station machines [--json] # All machines: every Mac, its VMs under it, then Needs you
station machines <machine> [--json] # that machine's Overview, as station overview --on <machine>
station machines add <mac> [--user <login>] [--key <file>]
station machines forget <mac> [--yes]

station machines create [<name>] [--persistent] [--image linux] [--cpus <n>] [--memory <size>] [--disk <size>] [--network nat|bridged]
station machines create <name> --server [--cpus <n>] [--memory <size>] [--disk <size>] [--check <url|host:port>]
station machines start <vm>
station machines stop <vm> [--yes | --confirm=<name>]
station machines restart <machine> [--yes]
station machines edit <vm> [--cpus <n>] [--memory <size>] [--disk <size>] [--check <url|host:port> | --no-check]
station machines ssh <machine>
station machines log <vm>
station machines cleanup [<vm>...] [--yes]
station machines delete <vm> [--confirm=<name>]
station machines unlock <mac>
station machines wake <mac> [--tell]
station machines remind <mac>
```

`--this-mac` lists This Mac only, without asking the other Macs. `unlock` is for a Mac waiting at FileVault after a power cut: a Mac that answers already says so (exit 0); otherwise it needs a terminal and dials the Mac on your LAN, at the address it last answered with (Tailscale starts only once its disk is unlocked), where you type the password into macOS's own pre-boot prompt, then waits for the Mac on Tailscale again. `wake` sends a Wake-on-LAN packet from this Mac to the hardware address the Mac's last answer said (a Mac that answers already says so, exit 0; one never heard has none, exit 1); `--tell` also tells you once, `mac_back`, the moment it answers again. `remind` is that ask alone (the menu bar's Tell me when it wakes): it sends nothing, so a Mac never heard can be asked too; one that answers already says so and nothing is armed. `restart <mac>` asks yes or no naming the VMs that pause, sends a planned restart that unlocks itself, waits for its station to answer again (`did not come back`, exit 1, past 3 minutes), starts those VMs again and says how they are. `create` makes an ephemeral workspace unless `--persistent` is given, or a server with `--server` (always on, named, behind its Mac, 4 CPU · 8 GB · 50 GB by default; `--check` is its health check, refused on a workspace; `--always-on` is its hidden v2 alias); it makes the VM only: `--task` and `--agent` are refused (exit 2) with the `station agents start "<task>" --new` that does it now; and asks the Mac on a terminal when two answer; piped it needs no question. A new VM's name is one no Mac, no VM and no agent of any machine holds, nor any agent's `·<n>` of it, the names the VM would give its own agents unasked (`"docs-bot" is taken (an agent on linux-1)`, exit 2); a create with no name takes the first `<image>-<n>` free so. `stop` and `delete` of a workspace that hosts live agents name them in their question, and stop them first, each finished, its terminal closed; one that does not stop is a warning under the outcome (`linux-1 did not stop: …`), and the VM goes on; `stop` asks a workspace yes or no and a server its name typed (`--confirm=<name>`; `--yes` never answers it), and the VM's own Mac judges the answer. `add` writes the Mac's ssh block, then reaches it as every later call will: a Mac that does not answer, or answers with no station, is not added and nothing stays written; one that runs another station version is added, with how to bring the two to one. `delete` is the only verb that destroys a machine; `forget` only unregisters a Mac, here, whatever `--on` says: it never reaches that Mac, and refuses This Mac and a VM. VMs are made and run on Macs, so these commands are refused on a VM. `create`, `start`, `stop`, `restart`, `edit` and `delete` need the Mac's engine (`Hint: station engine start`). `log` prints the lines a VM's runs wrote, its console's then its hypervisor's own, on the VM's own Mac. `cleanup` deletes, after one yes or no, every ephemeral workspace of the Mac that is stopped or whose agent finished (or only those named), each with its disk, its ports and its shared accounts, and names the VMs still running, which stay; naming one that may not go is refused before any goes. `station machines <machine>` is that machine's Overview (`probe` is its hidden v2 alias). `--network bridged` is for macOS VMs only. See [Machines](machines.md).

### `station agents`

```sh
station agents [--on <workspace>] [--json] # every agent of every machine, or that workspace's
station agents start "<task>" --on <workspace> [--name <name>] [--app <app>] [--folder <path>]
station agents start "<task>" --new [--mac <mac>] [--name <name>] [--image linux] [--cpus <n>] [--memory <size>] [--disk <size>]
station agents reply <agent> [<text>]
station agents attach <agent>
station agents merge <agent>
station agents stop <agent> [--yes]
station agents forget <agent>
```

An agent is a job on a workspace, one Herdr terminal on its Herdr machine ([ADR 024](../decisions/024-agents-run-in-herdr.md)); a workspace hosts any number of them, and an agent's lifecycle is its own. Each has a name, short and unique across every machine: lowercase letters, digits and `-`, then at most one `·<n>`. A workspace's first agent takes the workspace's name, the next ones `<workspace>·2`, `<workspace>·3`, the first no agent of any machine holds; `--name` gives another. A name is checked against every machine by the call that starts the agent, `agents.start`, whichever client asks it: on a Mac it reads every Mac that answers, and inside a workspace it asks its Mac's engine, which reads every Mac it knows; while that engine is down the workspace checks its own agents alone. A name any agent holds starts nothing (`an agent is already named docs-bot`, exit 2), nor does another workspace's own name or its `·<n>`, which that workspace gives its agents unasked, one run by hand included (`ci-1 is ci-1's: a workspace names its own agents ci-1 and ci-1·<n>`, exit 2). Every verb takes the name alone: on a Mac station reads every running workspace of every Mac that answers for it, and `--on`, or a run inside a workspace, looks there only. Should two agents still hold one name (a journal kept from before names were checked across machines), a verb by that name is refused, naming each with its workspace (`more than one agent is named docs-bot: docs-bot on linux-1 and docs-bot on ci-1`, exit 2), and `--on` picks one.

`station agents` lists the agents waiting on you first, with what they wait on, then those at work with their step, then the finished, each with its workspace, its Mac and since when; `--on` one workspace lists its own. `start` needs exactly one of `--on` (a running workspace) and `--new` (an ephemeral workspace made first with New VM's options, whose name the agent takes); neither, or both, is refused, and inside a workspace the agent starts on it: `--new` there is refused. Its task is typed as its first prompt once the agent is ready, never on a command line; without one it waits for a prompt. An ephemeral workspace whose agent does not start is kept (exit 1), with the `start --on` that starts it. `reply` types the text, or stdin when none is given, into the agent's terminal and the agent is working again; `attach` puts this terminal on it (`hc attach <workspace>/<terminal>`) until you detach (ctrl+\\); `merge` merges the PR it opened with the GitHub account shared with its workspace, without a question (owner decision D2), and it finishes. `stop` asks yes or no (`--yes`), closes its terminal and finishes it, reason `stopped`; its workspace keeps running. `forget` takes a finished agent off the list and out of its workspace's journal, its terminal too if Herdr still holds it; a live one is refused with the `stop` to run first.

## On a machine

These commands act on the target machine: the one `--on` names, else the one `station use` chose for this terminal, else This Mac. Each lists only what the machine's kind allows; a section its kind does not have is refused, exit 2, with the reason in words.

### `station overview`

```sh
station overview [--all] [--json]
```

This machine at a glance (SPEC §5.1): a first line that sums it up (`This Mac · jterrazz-macbook-pro · 5 things need you · Engine running`; a VM's names its kind and how long it has run), then Needs you, every section's findings ranked broken first and then by the one section order, five of them unless `--all`; then one line per section the machine has, in that order (a Mac: Activity, Apps, Settings, Disk, Ports, Accounts, Background, Awake, then its VMs; a workspace: the sections it has; a server: Activity, Ports and SSH, and `Apps, settings, accounts: not managed by station`); Activity's line is CPU, memory, free space and network in, from the engine's history when it spans a rate, else one sample now, and its findings join Needs you; then one or two `→` next steps. Each line is that section's own read, so its counts equal what the section's own command prints. `station overview --on <vm>` and `station machines <vm>` print the same bytes: `On <vm>:`, the VM's first line from its Mac (`linux-1 · Workspace on This Mac · Ephemeral · up 2 h`), the sections it reads inside it, then its Health; a VM that does not run, or whose guest does not answer, shows its first line and Health alone. `--json` prints `station.overview.read/v1`: `machine`, `findings[]` (each `fix` one of `call`, `command`, `path`) and `sections[]` with their counts; the hidden alias `station status --json` keeps printing the [`station.status/v1`](status-json.md) document `studio doctor` reads. On a terminal, `overview`, `apps` and `disk` draw every row at once and fill each as its read answers ([instant reads](../concepts/design-system.md#instant-reads)); piped or with `--json` they print one complete document.

### `station activity`

```sh
station activity [--all-machines] [--json [--history]]
```

What a machine is doing now: CPU and load, memory and its pressure, swap, free space, disk and network rates and totals, thermal and battery on a Mac, uptime, and the busiest processes; `--all-machines` is one row per machine and gathers every machine's findings (refused with `--on`, exit 2). `--json` prints `station.activity.read/v1`; `--history` adds the engine's 24 h series and needs `--json` (exit 2 without). Where each value comes from, and the history, are [Activity](activity.md)'s.

### `station apps`

```sh
station apps [<app>] [--all] [--json] # the apps for this machine, by attention; one app's detail
station apps install <app...>
station apps update [<app...> | --method <method>... | --all-machines]
station apps remove <app...> [--yes]
```

The list sums This Mac first (`108 apps for this Mac · 3 updates · 100 not installed`), then groups the apps by attention: Updates, Failed, Not installed, Not managed (`?`: present, but not installed by its catalogue method, the detail saying how it got there), and OK folded; the apps that live on other machines are a count, every row of every group unfolded with `--all`. `station apps <app>` is one app's detail: its method, its version and the newest that waits (`2.97.0 → 2.98.0`), where it lives, its category and what it brings. `--json` is `station.apps.list/v1`, each app with `name`, `category`, `method`, `state`, `installed` (`null` when it is not), `newest` (with an update only), `lives_on` (`every-machine`, `mac-only`, `vms-only`), `provides` and `detail`.

`install` installs in dependency order, each app with what it requires: a requirement missing here (an app it names, its method's manager) joins the run, so `apps install gh` on a Mac without Homebrew installs homebrew first. Then come the settings each app brings. On a terminal its spinner names what runs and why (`Installing homebrew · required by gh`); a step whose installer may ask on the terminal (one that asks your password, every curl installer) is a line of its own, `▸ Installing homebrew · required by gh · asks your password once`, so the prompt under it is never drawn over. Ctrl-c stops it with nothing printed after, exit 130, what finished staying installed. An installer's own words go to its app's log, `~/.local/state/station/logs/<app>.log`, never to the terminal; a failure names that file (`✗ claude  script  the install failed: its output is in ~/.local/state/station/logs/claude.log`). Its outcome opens with the count, the machine and the time (`✓ Installed 2 apps on This Mac · 9s`; `✗ 1 app failed to install on This Mac` when none did), then each app with the version and the method that hold it and who required it, then `→ exec zsh` once a manager's commands landed off this shell's PATH; the pipe keeps the app rows alone. An app already installed says so with its version, `● jq 1.8.1 brew already installed`, exit 0. A bare `update` updates only the apps with an update and says so the same way (`● Updated 3 apps on This Mac`; `● every app is up to date on This Mac` when none); `--method` takes `brew` (brew and cask), `npm`, `bun`, `uv`, `mise` or `skills`, never with names; `--all-machines` updates each Mac and running workspace in turn, says each server and stopped VM it skips, and exits 1 when a Mac does not answer. `remove` asks on a terminal and needs `--yes` off one; an app not installed is refused before anything runs.

The refusals exit 2: `no app named "jqq"`, `dia lives on Macs only` (hint `--on <mac>`), `lazygit is not installed`, `infra is a server: station does not manage its apps`; a requirement this machine cannot carry refuses the install the same way, by its own name. An agent skill is an app: `station apps install frontend-design`. The model is [Apps and skills](apps-and-skills.md).

### `station settings`

```sh
station settings [--all] [--json] # Set by station: applied, changed since, not applied, off here
station settings --all-machines [--all] [--json] # the matrix: one row per setting, one column per Mac and workspace
station settings <setting> [--json] # one setting's detail
station settings diff <setting> # what station would write against what is here
station settings apply [<setting...>] [--yes] [--all-machines] # bare = every setting not applied or changed since that asks nothing
station settings off <setting...> [--yes] # off here: station neither applies nor watches it on this machine
```

A setting that needs values asks them in a form on a terminal and is refused elsewhere (exit 2); a value is never an argument. An apply over a setting changed since asks first, as yours is lost; off removes only what is still exactly station's, a change of yours stays. The Always-on Mac group is applied by name only. `--all-machines` reads every Mac and workspace at once: first the machines (a stopped workspace shows its last read with its age, kept by This Mac's engine in its memory alone, so `?` once it restarted; a Mac that does not answer says when it last did; a server is named and has no column), then what needs a look with its gesture, then each group's rows with one mark per machine and their OK count, which leaves out what is not for its machine; the rows that need no look fold on one line unless `--all`. `--json` is `station.settings.matrix/v1`. `settings apply --all-machines` applies on each Mac and running workspace in turn, says each server and stopped workspace it skips, each setting named that does not live on a machine (skipped there alone, the rest applied), and each admin setting with the command that applies it on that Mac's own terminal, and exits 1 when a Mac does not answer. `station config`, `config install` and `config uninstall` are hidden aliases of `settings`, `settings apply` and `settings off` for one release. See [Settings](settings.md).

### `station disk`

```bash
station disk [--json] # the volume, then Misplaced, Can rebuild, Your data, Never touched, with sizes; nothing runs
station disk free [<app...>] # empty what an app can rebuild with its own command; bare = every Can rebuild row
station disk erase <app> [--confirm=<app>] # erase an app's data, back to factory; its name typed
station disk move <folder> # move a misplaced folder where its app expects it
```

`free` and `move` ask nothing: nothing is lost. `erase` is always one app, its name typed, and refuses while the app runs or on a Never touched row. A stale VM goes with `station machines delete`, an old image version with `station images delete`. The v2 `station storage`, `storage clean` and `storage reset` are hidden aliases for one release. `station disk free docker` prunes Docker's unused data. See [Disk](disk.md).

### `station ports`

```sh
station ports [--all-machines] [--all] [--json] # every listening port in its class: Needs you, Public, Private
station ports share <vm> <port> [--json]      # reserve the port now and share it; inside a VM: station ports share <port>
station ports stop-sharing <vm> <port> [--yes] # release it, and do not share it again until a share; inside a VM: <port>
station ports declare [<machine>] <port> --via <route> [--url <link>] # declare it public: cloudflare-tunnel, tailscale-funnel or lan
station ports undeclare [<machine>] <port> [--yes]     # take the declaration back
station ports https on|off [--on <mac>]               # HTTPS links on a Mac: https:// with a certificate, or http://
```

`--all-machines` asks every Mac through its engine; `--all` lists the ports that listen on their machine only, otherwise counted on one line. `share` prints the URL before anything listens, piped the URL alone; a port is refused on a VM that shares 20 already (exit 1, `Hint: station ports stop-sharing <vm> <port>`) and while This Mac is signed out of Tailscale (exit 1, the reservation kept). `stop-sharing` asks yes or no and is remembered in `preferences.json` (`ports.not_shared`). `declare` and `undeclare` are kept by the machine's Mac and refused inside a VM; the route is verified on every list (see [Ports](ports.md)). `--json` is `station.ports.list/v1`. The v2 nouns of this family and `tailnet` are hidden aliases for one release. See [Ports](ports.md).

#### The Tailscale line

The first line of `station ports` is This Mac's node: signed in with its name and whether Serve shares, signed out, or not installed. Read only: the owner logs in through the Tailscale app, and station never drives it but for [shared ports](ports.md#serve-and-https).

### `station accounts`

```sh
station accounts [--all-machines] [--json] # who has access to what here, then the shares with VMs
station accounts share <account> --with <vm> [--for <duration>] # 12h by default, 1m to 7d
station accounts take-back <account> --from <vm> [--yes]
```

This Mac's GitHub, npm, Claude, Tailscale, Cloudflare, git, ssh and Docker sign-ins, then `# Shared with VMs`: each share with its VM and its time left. `--on <vm>` reads a workspace's own sign-ins, by its guest; `--all-machines` prints one block per Mac and workspace, a server named and not listed, a stopped workspace and a Mac that does not answer each its own line. Identities and scopes only: no token, cookie or key body is ever read into a value, and anything shaped like one reads `?`. `--json` is `station.accounts.list/v1`: `{schema, machine, accounts, shares, share_list}`. A share is asked of the VM's own Mac. `station secrets`, `secrets lend --to` and `secrets recall --from` are hidden aliases of `accounts`, `accounts share` and `accounts take-back` for one release; `secrets allow` and `disallow` are retired (exit 2). See [Accounts](accounts.md#sharing-accounts).

### `station background`

```sh
station background [--all] [--json] # every non-Apple launchd job: Failing, Orphans, then Station, Known apps, Unknown folded
station background remove <label> [--yes] # a task of your own: boot it out and delete its plist
```

An orphan is a task whose program is missing or whose app is gone. `remove` asks on a terminal and needs `--yes` off one; it takes only a user-scope task: a root one is refused, exit 2, with the command to run yourself (station never runs sudo), and the engine's own job is refused with `station engine stop`. A Linux workspace has no job manager station reads, so `--on` one is refused; a macOS workspace lists its own launchd tasks. `--json` is `station.background.list/v1`.

### `station awake`

```sh
station awake [--json] # every Mac, this one first: its mode, its guards, its reading
station awake <for> [--lid] [--battery] # keep this Mac awake: 30m, 1h, 4h, until 18:00 (1 min to 24 h); held by the engine
station awake --until-done <vm> # keep this Mac awake until the agent working in that VM is done (24 h at most)
station awake off # end the keep-awake now; Never sleeps stays
```

A Mac only; a guard already past its threshold refuses the start (exit 1). Piped, one record per row: the session (`on`, its flags, until in RFC 3339, who asked), the guards, then the reading as `key<TAB>value`. The modes, the guards and Never sleeps are [Awake](awake.md)'s.

## A Mac's images and engine

### `station images`

```sh
station images [--json] # every Mac's, one row per version per Mac
station images build [<image...>] [--binary <linux station binary>]
station images update [<image...>] # the newest released version
station images delete <image>[@<version>] [--yes]
```

A bare `build` builds every image that is missing or whose recipe changed; a bare `update` updates every image this Mac holds. An image the catalogue does not hold is `no image "debian"` (exit 2). See [Images](images.md).

### `station engine`

```sh
station engine [--json] # running, its version, up, the VMs it runs, Awake; or stopped, since when and by what
station engine stop [--yes] # Stop everything on this Mac: every running workspace stopped, its disk kept; Awake and the sharing end; then the engine, unless a server runs: it is skipped, and the engine keeps running for it
station engine start # the engine installed when it is not, restarted when it runs another version, bootstrapped again; the VMs it stopped stay stopped
```

`--json` is `station.engine.read/v1`: `{schema, engine: {state, version, up_s, vms, stopped_at, stopped_by}}`, `state` `running`, `stopped` or `not-installed`. An engine running another version than station names both and `station engine start`. A Mac only; `--on <mac>` runs it on another Mac. `station stop` and `station start` are hidden aliases of `engine stop` and `engine start` for one release, saying `→ now: station engine stop` on stderr (see [Hidden aliases](#hidden-aliases)). A start the job manager refuses ends in `the engine did not start: <its last word>` (exit 1). See [Machines](machines.md#the-engine).

## This terminal

### `station use`

```sh
station use <machine> # every later command of this terminal runs there, as --on would
station use - # back to the machine used before
station use # the machine this terminal uses, and since when
```

`--on` still wins for its one command; other terminals stay where they are. See [Machines](machines.md#this-terminals-machine).

## Station itself

```sh
station init [--name <name>]
station --version
station help [command]
```

`init` makes this machine known to station, safe to run again: it names this Mac (asking the name on a terminal, taking `--name` otherwise; off a terminal, a Mac with no name yet is refused, exit 2, `Hint: station init --name <name>`), writes the one managed block of `~/.zshrc`, moves `~/.studio/` to the XDG folders and brings every store to its version, and writes the apps' own files (`relocate`). It installs no app, runs no package manager and asks no password; the engine comes with the first VM command on a Mac with tart, or `station engine start`. On a terminal it opens on a line that sums it up and ends on the empty state of what the machine carries (`Machinefile  none yet: apps and settings come one at a time`), then at most two next steps: `exec zsh` when it wrote the block, `station apps install <app>`, `station settings apply <setting>`. Inside a workspace it writes the block and the layout only; inside a server it sets nothing up, the server's own tooling keeps it. `--version` prints `station <version> (<commit>)`. See [Getting started](../start/getting-started.md) and [Dotfiles](dotfiles.md).

## Hidden aliases

A v2 word, or a v3.0 one (`setup`), runs its new command with the same arguments, for one release, and prints one muted line on stderr naming it (nothing there with `--json` or `--page`); help and completion never list it.

| Old                                   | Runs                                                                                                                                                                                                      |
| ------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `station status`                      | `station overview`; `status --json` keeps `station.status/v1`                                                                                                                                             |
| `station stop`                        | `station engine stop`                                                                                                                                                                                     |
| `station start`                       | `station engine start`                                                                                                                                                                                    |
| `station tools`                       | `station apps`; `install`, `uninstall` and `upgrade --via` run `apps install`, `apps remove` and `apps update --method`                                                                                   |
| `station secrets`                     | `station accounts`; `lend --to` and `recall --from` run `accounts share --with` and `accounts take-back --from`                                                                                           |
| `station config`                      | `station settings`; `install` and `uninstall` run `settings apply` and `settings off`                                                                                                                     |
| `station storage`                     | `station disk`; `clean` and `reset` run `disk free` and `disk erase`                                                                                                                                      |
| `station power`                       | `station awake`; `awake --for <d>` runs `awake <d>`, `sleep` runs `awake off`                                                                                                                             |
| `station services`, `station tailnet` | `station ports`; `url` and `release` run `ports share` and `ports stop-sharing`                                                                                                                           |
| `station machines probe <machine>`    | `station machines <machine>`                                                                                                                                                                              |
| `station setup`                       | `station init`, v3.0's word, and one more muted line on stderr: apps and settings no longer come with it (`station apps install <app>`, `station settings apply <setting>`)                               |
| `machines create --always-on`         | `machines create --server`                                                                                                                                                                                |
| `machines create --permanent`         | `machines create --persistent`; a lifetime stored or sent as `throwaway` or `permanent` reads as `ephemeral` or `persistent`, and the store is rewritten on the next `station init`                       |
| `station run git <shortcut>`          | `git <alias>`, the [Git shortcuts](settings.md#the-items) setting's alias (`log`, `push`, `status` are `lg`, `pushup`, `st`); refused with `station settings apply git-shortcuts` while it is not applied |
| `station run docker rm\|rmi\|reset`   | the steps of `dclean`, `dclean-images`, `dreset`, the [Docker shortcuts](settings.md#the-items), asking as before (`--yes` off a terminal)                                                                |

The words of a kind and a lifetime are not aliased in `--json`: `station.machines.list/v1`, `station.accounts.list/v1` and `station.overview.read/v1` say `computer`, `ephemeral` and `persistent` where they said `mac`, `throwaway` and `permanent`, and `station.status/v1` keeps its words ([ADR-032](../decisions/032-the-words-of-a-vms-lifetime-and-network.md)).

## Plumbing

These commands exist for the machine's own use. They are hidden from help and the owner never types them.

| Command                             | What runs it and what it does                                                                                                                                                                                                                        |
| ----------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `station shell zsh`                 | the `~/.zshrc` block evaluates it at every shell start: PATH, the mise activation, station's completion, and in a VM the shared Claude token and the [profiles'](ports.md#profiles) exports. Output on stdout only; another shell or none is refused |
| `station shell prompt`              | the prompt's `custom.station` segment runs it: `on linux-1` while this terminal [uses](machines.md#this-terminals-machine) another machine, nothing otherwise; it never counts as a use                                                              |
| `station completion <shell>`        | cobra's completion script, for a shell the block does not serve                                                                                                                                                                                      |
| `station engine run`                | the launchd agent `com.jterrazz.station.engine` runs it on a Mac: the VMs and their ports ([Machines](machines.md#the-engine))                                                                                                                       |
| `station guest serve --stdio`       | the Mac's engine runs it inside each VM over its one guest connection: the VM's calls, and what listens there ([The API](api.md#the-transport))                                                                                                      |
| `station agents record --app <app>` | an agent's hook runs it inside a workspace on each event, the event on stdin or as its last argument: it folds it into the VM's journal of agents, prints nothing, and exits 0 within 200 ms whatever its flags ([Settings](settings.md#the-items))  |
| `station engine attach`             | another Mac runs it over ssh: bridges ssh's stdio to this Mac's engine socket ([The API](api.md#the-transport))                                                                                                                                      |
| `station machines pipe <vm>`        | a VM's `~/.ssh/config` ProxyCommand runs it: ssh's bytes to the VM's ip behind its Mac ([Machines](machines.md#ssh-and-herdr))                                                                                                                       |
| `station machines supervise <vm>`   | retired: an old per-VM agent that still runs it exits at once; `station engine start` removes those agents                                                                                                                                           |

`--on` is refused on `shell`, `completion`, `help`, `use`, `engine run` (and `daemon`, its v2 name), `engine attach`, `guest`, `machines pipe` and `machines supervise`: none of them sends a call.

## Exit codes

| Code | Means                                                                                                                 |
| ---- | --------------------------------------------------------------------------------------------------------------------- |
| 0    | Done, or a read that found findings                                                                                   |
| 1    | `failed`, `unreachable`, `engine-stopped`, `far-too-old`, `declined`: an operation failed, or you said no             |
| 2    | `refused`, `not-found`, `not-for-this-machine`, `needs-confirmation`, `needs-terminal`: before any question or effect |
| 130  | `interrupted`; nothing is printed                                                                                     |

The exit code is computed once, from the error's type ([Architecture](../concepts/architecture.md#the-error-model)). A command over several items prints every outcome, then exits 1 with `<n> of <m> <noun> failed` if any failed.

## Related

- [Design system](../concepts/design-system.md): the forms a document takes.
- [Status contract](status-json.md): the v2 JSON the hidden `status --json` keeps for one release.
- [The API](api.md): the calls behind every command.
