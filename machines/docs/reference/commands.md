# Commands

Every command of `station`, grouped as `station help` groups them: the four groups of [the grammar](../../../docs/reference/cli.md#the-help), of which the OS group holds no command yet, so the help does not draw it. The words are the [Glossary](../concepts/glossary.md)'s.

| Section    | Answers                                                                                                                       |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------- |
| Rules      | What holds for every command                                                                                                  |
| Machine    | `machines`, `devices`, `images`, `engine`, `apps`, `settings`, `disk`, `ports`, `accounts`, `background`, `awake`, `activity` |
| Agent      | `agents`, `skills`, `project`, `teams`, `automations`                                                                         |
| Spwn       | `overview`, `init`, `--version`, `use`, `journal`, `machinefile`, `help`                                                      |
| Plumbing   | The hidden commands the machine runs for itself                                                                               |
| Exit codes | What 0, 1, 2 and 130 mean                                                                                                     |

## Rules

- Bare `station` prints `station overview`, with its `--all` and `--json`, exit 0: this machine at a glance, then the next steps as commands.
- A bare noun lists (`apps`, `settings`, `disk`, `machines`, `ports`, `images`, `accounts`, `machinefile`). A verb whose argument is missing is refused, its hint the command with it: `station apps install` hints `station apps install <app>`.
- An existing thing is a positional (`machines stop api`); a new thing is described by options (`machines create api --persistent`).
- One verb per meaning, the [Glossary](../concepts/glossary.md#the-verbs)'s: `apps` install, update, remove; `settings` apply, diff, off; `disk` free, erase, move; a VM create, start, stop, restart, edit, delete.
- Every read takes `--json`. Every destructive action asks on a terminal, and what answers it in a script is the question's: `--yes` for a yes or no (`images delete`, `apps remove`), `--confirm=<name>` for one that asks the name typed (`machines delete`, `disk erase`), neither where nothing is asked (`images build`). A `--yes` given to a typed question or to none is refused, exit 2, naming which it met; `--yes` never answers a typed one.
- Every refusal exits 2 before any question; every failure is one `Error:` and one `Hint:`. With `--json`, the same refusal or failure is also the `station.error/v1` document on stdout (`schema`, `type`, `code`, `subject`, `message`, `hint`, `fix`), the exit code unchanged. A question answered no is `Error: <what> kept: you said no`, exit 1.
- A command that needs an app this Mac lacks is refused before anything runs, exit 2, naming the app and what needs it: `Error: tart is required by machines`, `Hint: station apps install tart`. The needs are the [Glossary](../concepts/glossary.md)'s: the VM verbs that make or run a VM and the image verbs need tart, `ports share` needs Tailscale. On a terminal the refusal is one line, `○ tart is required by machines · station apps install tart`, then `Install tart now?`: yes installs it as `apps install` does and runs the command again; no ends there, exit 2. A need Machines installs none of (the studio CLI) is the refusal on a terminal too.
- A word Machines does not know is `Error: unknown command "<word>"`, exit 2, and its hint is the command it meant (`Hint: station machines` for `station machnies`, `Hint: station machines delete` under a noun), else `station help`; so is `station help <word>`. A machine or app name it does not know says the name it meant (`Hint: Did you mean linux-1?`). `station apps search <app>` answers `station apps <app>`.
- A run with `--json` or `--page` never asks: a question it would put is refused, exit 2, naming the flag that answers it (`--yes`, `--confirm=<name>`, `--on` for the Mac a create picks).
- A word under `disk` that names one of its entries, the verb forgotten, is still refused, but its hint is the command meant: `station disk go` is `"go" is a disk row: say what to do with it`, hint `station disk free go`. Under `apps` the entry is a read: `station apps jq` is jq's detail; under `settings`, a setting's name is its read: `station settings dns`.
- `--on <machine>` runs any command below on another machine, through its engine, and renders the answer on this one; see [Machines](machines.md#on-another-machine).
- `station use <machine>` makes every later command of this terminal run there, as `--on` would; `--on` still wins for one command. See [Machines](machines.md#this-terminals-machine).
- A value a command needs from you is never an argv flag when it could be a secret: a setting asks in a form on a terminal ([Settings](settings.md)).

## Machine

These commands act on the target machine: the one `--on` names, else the one `station use` chose for this terminal, else This Mac. Each lists only what the machine's kind allows; a section its kind does not have is refused, exit 2, with the reason in words.

### `station machines`

```sh
station machines [--json] # All machines: every Mac, its VMs under it, then Needs you
station machines <machine> [--json] # that machine's Overview, as station overview --on <machine>
station machines add <mac> [--user <login>] [--key <file>]
station machines forget <mac> [--yes]

station machines create [<name>] [--persistent] [--image base-ubuntu | --from <machinefile>] [--for <purpose>] [--cpus <n>] [--memory <size>] [--disk <size>] [--network nat|bridged]
station machines create <name> --server [--for <purpose>] [--cpus <n>] [--memory <size>] [--disk <size>] [--check <url|host:port>]
station machines start <vm>
station machines stop <vm> [--yes | --confirm=<name>]
station machines restart <machine> [--yes | --confirm=<name>]
station machines edit <vm> [--cpus <n>] [--memory <size>] [--disk <size>] [--check <url|host:port> | --no-check]
station machines ssh <machine>
station machines log <vm>
station machines cleanup [<vm>...] [--yes]
station machines delete <vm> [--confirm=<name>]
station machines unlock <mac>
station machines wake <mac> [--tell]
station machines remind <mac>
```

`--this-mac` lists This Mac only, without asking the other Macs. `unlock` is for a Mac waiting at FileVault after a power cut: a Mac that answers already says so (exit 0); otherwise it needs a terminal and dials the Mac on your LAN, at the address it last answered with (Tailscale starts only once its disk is unlocked), where you type the password into macOS's own pre-boot prompt, then waits for the Mac on Tailscale again. `wake` sends a Wake-on-LAN packet from this Mac to the hardware address the Mac's last answer said (a Mac that answers already says so, exit 0; one never heard has none, exit 1); `--tell` also tells you once, `mac_back`, the moment it answers again. `remind` is that ask alone (the menu bar's Tell me when it wakes): it sends nothing, so a Mac never heard can be asked too; one that answers already says so and nothing is armed. `restart <mac>` asks yes or no naming the VMs that pause, sends a planned restart that unlocks itself, waits for its station to answer again (`did not come back`, exit 1, past 3 minutes), starts those VMs again and says how they are. `create` makes an ephemeral workspace unless `--persistent` is given, or a server with `--server` (always on, named, behind its Mac, 4 CPU · 8 GB · 50 GB by default; `--check` is its health check, refused on a workspace); and asks the Mac on a terminal when two answer; piped it needs no question. An image not built on that Mac is built first, with no question, and the create says so on stderr before the build starts (`base-ubuntu is built first, which takes several minutes`). A new VM's name is one no Mac and no VM of any machine holds; a create with no name takes the first `<image>-<n>` free so. `stop` asks a workspace nothing and prints how to start it again (`station machines start <vm>`), and asks a server one yes or no that says what it serves stops (`--yes`, or its name with `--confirm=<name>`); the VM's own Mac judges the answer ([ADR-097](../decisions/097-a-confirmation-follows-its-consequence.md)). `add` writes the Mac's ssh block, then reaches it as every later call will, then writes the blocks of the VMs every Mac now holds, the new one's included: a Mac that does not answer, or answers with no station, is not added and nothing stays written; one that runs another station version is added, with how to bring the two to one. `delete` is the only verb that destroys a machine; `forget` only unregisters a Mac, here, whatever `--on` says: it never reaches that Mac, and refuses This Mac and a VM. VMs are made and run on Macs, so these commands are refused on a VM. `create`, `start`, `stop`, `restart`, `edit` and `delete` need the Mac's engine (`Hint: station engine start`). `log` prints the lines a VM's runs wrote, its console's then its hypervisor's own, on the VM's own Mac. `cleanup` deletes, after one yes or no, every ephemeral workspace of the Mac that is stopped (or only those named), each with its disk, its ports and its grants, and names the VMs still running, which stay; naming one that may not go is refused before any goes. `station machines <machine>` is that machine's Overview. `--network bridged` is for macOS VMs only. See [Machines](machines.md).

### `station devices`

```sh
station devices [--json] # this Mac's own key, then the other devices it lets into its VMs
station devices add <name> <public key> [--channels spwn-api,pipe,herdr]
station devices forget <name> [--confirm=<name>]
```

The device book of this Mac, or with `--on` another Mac's; it reads with the engine down. `add` asks for you at this Mac every time and is refused from another machine (`devices.add-here-only`, exit 2); refused or with nobody to ask, it exits 1 and writes nothing. `forget` asks no presence and works with `--on`. An entry whose seal broke, or every entry of a book that is not the one this Mac last wrote, is listed apart with the command that adds it again. `--json` prints `station.devices.list/v1`. A VM keeps no book, so it is refused there. See [Machines](machines.md#the-device-book).

### `station images`

```sh
station images [--json] # every Mac's, one row per version per Mac
station images build [<machinefile...>] [--binary <linux station binary>]
station images update [<image...>] # the newest released version
station images delete <image>[@<version>] [--yes]
```

An image is named after the Machinefile it is built from (`base-ubuntu`, `studio-linux`); `linux`, `station-linux` and `macos` name no image. A bare `build` builds every image that is missing or whose recipe, Machinefile or lock changed; a bare `update` updates every image this Mac holds. A name no source holds a Machinefile of, and no record of, is `no image "debian"`, and a Machinefile that makes no image is refused naming why (both exit 2). See [Images](images.md).

### `station engine`

```sh
station engine [--json] # running, its version, up, the VMs it runs, Awake; or stopped, since when and by what
station engine stop [--yes] [--remove] # Stop everything on this Mac: every running workspace stopped, its disk kept; Awake and the sharing end; then the engine, unless a server runs: it is skipped, and the engine keeps running for it; `--remove` then takes the engine's job from the job manager too
station engine start # the engine installed when it is not, restarted when it runs another version or serves no ssh agent, bootstrapped again; the VMs it stopped stay stopped
station engine restart # restart the engine's job, whatever it runs: the VMs that run keep running, the engine adopts them again, and with Spwn.app the menu bar restarts with it; it says the version and how many VMs run
```

`--json` is `station.engine.read/v1`: `{schema, engine: {state, version, up_s, vms, stopped_at, stopped_by}}`, `state` `running`, `stopped` or `not-installed`. An engine running another version than station names both and `station engine start`. `--remove` ends the engine's life on this Mac once it has stopped: without Spwn.app it deletes both agents' plists and boots them out, with it it unregisters both bundled agents from SMAppService, and `station engine` then reads `not-installed`; it removes nothing else ([Getting started](../start/getting-started.md#5-remove-what-it-put-down) lists the rest), and an engine kept running for a server keeps its jobs. A Mac only; `--on <mac>` runs it on another Mac. A start the job manager refuses ends in `the engine did not start: <its last word>` (exit 1). See [Machines](machines.md#the-engine).

### `station approvals`

```sh
station approvals [--json] # what macOS lets Spwn do on this Mac, one row per approval, as its engine read it
```

A Mac only, read from its engine, so it needs the engine running (`station engine start`), and with `--on` another Mac's. Each row is an approval (Local Network first), its state (`approved`, `refused`, `checking` while a refusal is not yet macOS's verdict, `unknown` where the engine cannot tell) and what it lets Spwn do; a refused one ends with where to approve it. None stops Spwn from running. `--json` is `station.approvals.list/v1`: `{schema, mac, rows}`, each row `{id, state, holder, required, sentence, fix}` ([API](api.md)).

### `station apps`

```sh
station apps [<app>] [--all] [--json] # the apps for this machine, by attention; one app's detail
station apps install <app...>
station apps update [<app...> | --manager <manager>... | --all-machines]
station apps remove <app...> [--yes]
station apps move <app> --to <manager>
```

The list sums This Mac first (`108 apps for this Mac · 3 updates · 100 not installed`), then groups the apps by attention: Updates, Failed, Not installed, Not managed (`?`: present, but installed by none of its catalogue methods, the detail saying how it got there), and OK folded; the apps that live on other machines are a count, every row of every group unfolded with `--all`. `station apps <app>` is one app's detail: its method, its version and the newest that waits (`2.97.0 → 2.98.0`), where it lives, its category and what it brings. `--json` is `station.apps.list/v1`, each app with `name`, `category`, `method`, `state`, `installed` (`null` when it is not), `newest` (with an update only), `lives_on` (`every-machine`, `mac-only`, `vms-only`), `provides` and `detail`.

`install` installs in dependency order, each app with what it requires: a requirement missing here (an app it requires, its method's provider, its method's manager, as the catalog's packages link them, the links a Machinefile's resolution follows) joins the run, so `apps install gh` on a Mac without Homebrew installs homebrew first. Then come the settings each app brings. On a terminal its spinner names what runs and why (`Installing homebrew · required by gh`); a step whose installer may ask on the terminal (one that asks your password, every curl installer) is a line of its own, `▸ Installing homebrew · required by gh · asks your password once`, so the prompt under it is never drawn over. Ctrl-c stops it with nothing printed after, exit 130, what finished staying installed. An installer's own words go to its app's log, `~/.local/state/station/logs/<app>.log`, never to the terminal; a failure names that file (`✗ claude  script  the install failed: its output is in ~/.local/state/station/logs/claude.log`). Its outcome opens with the count, the machine and the time (`✓ Installed 2 apps on This Mac · 9s`; `✗ 1 app failed to install on This Mac` when none did), then each app with the version and the method that hold it and who required it, then `→ exec zsh` once a manager's commands landed off this shell's PATH; the pipe keeps the app rows alone. An app already installed says so with its version and the method that holds it, `● jq 1.8.1 brew already installed`, exit 0. A bare `update` updates only the apps with an update and says so the same way (`● Updated 3 apps on This Mac`; `● every app is up to date on This Mac` when none); `--manager` takes a key a manager package of the catalog installs by (`brew`, `cask`, `mas`, `npm`, `bun`, `uv`, `mise`, `skill`), each reaching every key its manager installs by (`brew` and `cask` are both Homebrew's), never with names; `--all-machines` updates each Mac and running workspace in turn, says each server and stopped VM it skips, and exits 1 when a Mac does not answer. `remove` asks on a terminal and needs `--yes` off one; an app not installed is refused before anything runs. `move` switches an installed app to another of its methods here, `--to` naming it as a Machinefile's `manager:` does (`brew`, `mise`...): it installs by that one first, then uninstalls the copy the old method held, so the command never goes missing in between (`● jq  1.0.0  mise  moved from brew`, `● brew copy  uninstalled`); a copy no method holds (on PATH by hand) is yours and kept, one its method cannot uninstall is kept and said, a failed install leaves the old copy as it was, and an app already held by that method says so. A manager the app has no method by here is refused, exit 2, naming the ones it has (`jq has no method mise here, only brew`), its hint a move to one that does not hold it already, else the app's detail (`station apps jq`), as are an unknown manager, a missing `--to` and an app not installed.

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

A setting that needs values asks them in a form on a terminal and is refused elsewhere (exit 2); a value is never an argument. An apply over a setting changed since asks first, as yours is lost; off removes only what is still exactly Machines', a change of yours stays, and it exits 1 when any setting it turns off ends broken, a change of yours that still acts among them (an `authorized-key` block you changed, kept, whose key still signs in). The Always-on Mac group is applied by name only. `--all-machines` reads every Mac and workspace at once: first the machines (a stopped workspace shows its last read with its age, kept by This Mac's engine in its memory alone, so `?` once it restarted; a Mac that does not answer says when it last did; a server is named and has no column), then what needs a look with its gesture, then each group's rows with one mark per machine and their OK count, which leaves out what is not for its machine; the rows that need no look fold on one line unless `--all`. `--json` is `station.settings.matrix/v1`. `settings apply --all-machines` applies on each Mac and running workspace in turn, says each server and stopped workspace it skips, each setting named that does not live on a machine (skipped there alone, the rest applied), and each admin setting with the command that applies it on that Mac's own terminal, and exits 1 when a Mac does not answer. See [Settings](settings.md).

### `station disk`

```bash
station disk [--all] [--json] # the volume, then Misplaced, Can rebuild, Your data, Never touched, with sizes, then every VM disk the hypervisor holds; nothing runs
station disk free [<app...>] # empty what an app can rebuild with its own command; bare = every Can rebuild row
station disk erase <app> [--confirm=<app>] # erase an app's data, back to factory; its name typed
station disk move <folder> # move a misplaced folder where its app expects it
```

`--all` unfolds every group on a terminal, as `station background --all` does: a pipe lists every row already, so it changes nothing there. `free` and `move` ask nothing: nothing is lost. `erase` is always one app, its name typed, and refuses while the app runs or on a Never touched row. A stale VM goes with `station machines delete`, an old image version with `station images delete`. `station disk free docker` prunes Docker's unused data. See [Disk](disk.md).

### `station ports`

```sh
station ports [--all-machines] [--all] [--json] # every listening port in its class: Needs you, Public, Private, This machine only
station ports share <vm> <port> [--to <audience>] [--json] # reserve the port now and share it, with local, private or public; inside a VM: station ports share <port>
station ports stop-sharing <vm> <port> # release it, and do not share it again until a share; inside a VM: <port>
station ports declare [<machine>] <port> --via <route> [--url <link>] # declare it public: cloudflare-tunnel, tailscale-funnel or lan
station ports undeclare [<machine>] <port>           # take the declaration back
station ports https on|off [--on <mac>]               # HTTPS links on a Mac: https:// with a certificate, or http://
```

`--all-machines` asks every Mac through its engine; `--all` lists the ports that listen on their machine only, otherwise counted on one line. `share` prints the URL before anything listens, piped the URL alone; a port is refused on a VM that shares 20 already (exit 1, `Hint: station ports stop-sharing <vm> <port>`) and while This Mac is signed out of Tailscale (exit 1, the reservation kept). `stop-sharing` asks nothing, since a share gives the port back, and is remembered in `preferences.json` (`ports.not_shared`). `declare` and `undeclare` are kept by the machine's Mac and refused inside a VM; the route is verified on every list (see [Ports](ports.md)). `--json` is `station.ports.list/v1`. See [Ports](ports.md).

#### The Tailscale line

The first line of `station ports` is This Mac's node: signed in with its name and whether Serve shares, signed out, or not installed. Read only: the owner logs in through the Tailscale app, and Machines never drives it but for [shared ports](ports.md#serve-and-https).

### `station accounts`

```sh
station accounts [--all-machines] [--json] # who is signed in on this machine, and how
station accounts login <program> [--on <machine>] # sign a program in: its code or link is printed, you finish at its site
station accounts logout <program> [--on <machine>] [--yes] # sign a program out by its own logout; its token stays valid at its provider
station accounts add <program> [--label <name>] [--kind <kind>] [--from-sign-in] # keep a password in this Mac's vault, typed in a hidden field or taken from its sign-in
station accounts grant <entry> --on <vm> # give a password of the vault to a VM until it is removed; Touch ID asks first
station accounts remove <entry> [--on <vm>] [--yes] # on a VM its grant ends; on a Mac the password leaves the vault with every grant made from it
```

One row per program Machines signs in for (GitHub, npm, Claude, Cloudflare, Codex): signed in, signed out or broken, who, its scopes, and how, `own` or `granted` with the password's label. It is `accounts.list` of the machine; `--on <vm>` reads a workspace through the engine of its Mac, which knows what it granted. `--all-machines` prints one block per Mac and workspace, a server named and not listed, a stopped workspace and a Mac that does not answer each its own line. Identities and scopes only: no token, cookie or key body is ever read into a value. `--json` is `station.accounts.list/v1`: `{schema, machine, findings, accounts}`. `login` is `accounts.login` at that machine, a VM of a Mac through its Mac's engine: it prints each step as the engine words it (the heading, the code when there is one, the site, and the line that says what it waits for), then the program's row, exit 0; a refusal prints its words and exits non-zero (`login.unknown-program`, `login.by-grant` for a program signed in only by a grant, `login.already-signed-in`, `login.not-installed`, `login.expired`, `login.failed`; a server, `section.not-for-this-machine`), and Ctrl-C cancels the login. It waits on you, so off a terminal it is refused, exit 2, `Hint: run it in a terminal`; it opens no browser, and the site is the one you open. `logout` is `accounts.logout` at that machine and asks first (`--yes` off a terminal), then prints the program's row. `add` is `vault.add` of this Mac: a typed password's value is asked in a hidden field and never read from an argument, so off a terminal it is refused, exit 2, and `--from-sign-in` takes the token of this Mac's own sign-in to the program instead, asking nothing. An `<entry>` is a password of the vault by its label or its id; one that names none is `vault.unknown`, exit 2. `grant` is `grants.add` of the VM's Mac, and `remove` is `grants.remove` on a VM, `vault.remove` on a Mac, each asking first. See [Accounts](accounts.md).

### `station background`

```sh
station background [--all] [--json] # every non-Apple launchd job: Failing, Orphans, then Machines, Known apps, Unknown folded
station background remove <label> [--yes] # a task of your own: boot it out and delete its plist
```

An orphan is a task whose program is missing or whose app is gone. A job of Machines' own labels whose running binary is gone, or lies outside the Spwn.app this station runs from, is an orphan too, `not this Spwn's` with the binary it runs, removed by hand with the command it names, `launchctl bootout gui/<uid>/<label>`. `remove` asks on a terminal and needs `--yes` off one; it takes only a user-scope task: a root one is refused, exit 2, with the command to run yourself (Machines never runs sudo), and the engine's own job is refused with `station engine stop`. A Linux workspace has no job manager Machines reads, so `--on` one is refused; a macOS workspace lists its own launchd tasks. `--json` is `station.background.list/v1`.

### `station awake`

```sh
station awake [--json] # every Mac, this one first: its mode, its guards, its reading
station awake <for> [--lid] [--battery] # keep this Mac awake: 30m, 1h, 4h, until 18:00 (1 min to 24 h); held by the engine
station awake off # end the keep-awake now; Never sleeps stays
```

A Mac only; a guard already past its threshold refuses the start (exit 1). Piped, one record per row: the session (`on`, its flags, until in RFC 3339, who asked), the guards, then the reading as `key<TAB>value`. The modes, the guards and Never sleeps are [Awake](awake.md)'s.

### `station activity`

```sh
station activity [--all-machines] [--json [--history]]
```

What a machine is doing now: CPU and load, memory and its pressure, swap, free space, disk and network rates and totals, thermal and battery on a Mac, uptime, and the busiest processes; `--all-machines` is one row per machine and gathers every machine's findings (refused with `--on`, exit 2). `--json` prints `station.activity.read/v1`; `--history` adds the engine's 24 h series and needs `--json` (exit 2 without). Where each value comes from, and the history, are [Activity](activity.md)'s.

## Agent

These commands act on the agents of a project and on the skill files they are made of: the project of the folder a command runs in, found by its `spwn.yaml` from that folder up, or the user's own agents outside any project. The words are the [grammar's](../../../docs/reference/cli.md).

### `station agents`

```sh
station agents [<agent>] [--json] # the agents by name, their team and harness; one agent's Mind: its folder, its layers, its newest journal entries
station agents create <name> [--team <team>] [--harness <harness>] # a blank Mind
station agents edit <agent> [--team <team>] [--harness <harness>] # an empty value clears the field
station agents delete <agent> [--confirm=<agent>] # its Mind, every layer of it, for good
station agents fork <agent> --as <name> # a new agent with a copy of every layer, its journal left behind
station agents export <agent> [--to <folder>] # one archive, <agent>.tar.gz, the Mind left as it is
station agents import <archive> [--as <name>] # an agent from an archive, named as the archive
station agents dream <agent> # its journal read into a playbook of its Mind
station agents sleep <agent> # stale playbooks archived, old sessions pruned, the day logged
```

A name is lowercase letters, digits and dashes, a letter first, and no verb of `agents`; a name taken, a harness the catalog does not hold, and an archive no export wrote are refused, exit 2. `delete` asks the agent's name typed on a terminal and needs `--confirm=<agent>` off one. With `--on` a VM of this Mac, each command acts on that VM's agents: the project of this folder is the guest folder a share of the VM's start puts it at, and a project no share reaches, or `--on` another Mac, is refused, exit 2, with the Machinefile's `vm.mounts` to add it to. `--json` is `station.agents.list/v1`, or `station.agents.show/v1` for one agent.

### `station skills`

```sh
station skills [<skill>] [--json] # the skill files by name, with their description; one skill's file, its text as written
station skills create <name> # a markdown file made from a template, spwn/skills/<name>.md
station skills delete <skill> [--confirm=<skill>] # its file, for good
```

A skill is a bare markdown file, `spwn/skills/<name>.md` in the project, or `~/.spwn/skills/<name>.md` outside one; its description is the `description:` line of its frontmatter. A name is lowercase letters, digits and dashes, a letter first, and no verb of `skills`; a name taken is refused, exit 2. A skill's text is changed by opening its file: no command edits it. `delete` asks the skill's name typed on a terminal and needs `--confirm=<skill>` off one. `--on` reads the project as `station agents` does. `--json` is `station.skills.list/v1`, or `station.skills.show/v1` for one skill.

### `station project`

```sh
station project [--json] # the project of this folder: its spwn.yaml and its agents
station project create <name> [--harness <harness>] # a new folder, ./<name>, with its spwn.yaml and the starter agent neo
station project create [<name>] --from <example> # a new folder, ./<name> (default: the example's name), holding an example of the catalog
station project plan [--json] # what the project's rules find in its files, the errors first
station project drift [--json] # each reference spwn.lock does not record
station project apply [--yes] [--json] # render each agent into its run folder, .spwn/agents/<agent>/
```

A project is the files of a folder: the one whose `spwn.yaml` sits in this folder or the nearest above it, and a folder in none is refused, exit 2, with `station project create <name>` to make one. `create` makes `./<name>` and refuses a name that exists there, a link included, a name that is not lowercase letters, digits and dashes, a letter or a digit first, and a harness the catalog does not hold, exit 2; nothing is written outside the new folder. Its harness, `claude-code` unless `--harness` names another, is written in the starter agent's `agent.yaml`.

`create --from` makes the folder of one of the catalog's six examples (`macrohard`, `matrix`, `paperclip-factory`, `research-lab`, `severance`, `startup`; `station project create --help` names them): its `spwn.yaml`, `spwn.lock` and `spwn/` are copied as they are, the manifest's `name` set to the folder's, and `.spwn/` added to `.gitignore`. An example no one holds is refused with the nearest names, and `--harness` is refused with `--from`, exit 2, as each example agent declares its own. The example is the files of a project, so it plans valid and applies like any.

`plan` reads and writes nothing: the declaration's rules, then a render of each agent for the harness it declares (its own `runtime.backend`, else the project's, else `claude-code`) that is kept in memory, and, when it has no error, what `apply` would add, overwrite and remove under `To apply`. A plan with an error prints every issue, then exits 2; warnings alone exit 0. A `spwn:<name>` reference is not checked against a catalog. `drift` lists what was installed apart from what the project declares, and a row refuses nothing. Every command reads this folder in this process, so `--on` is refused. `--json` is `station.project.show/v1`, `station.project.plan/v1`, `station.project.drift/v1` and `station.project.apply/v1`.

`apply` makes the plan's `To apply` true, and nothing else: it lists it first, then writes each agent's rendered files into its run folder, `.spwn/agents/<agent>/`, the project's gitignored state, and keeps what it wrote in `.spwn/applied.json`. A plan with an error is printed and refused, exit 2, and nothing is written. An apply that only adds asks nothing; one that overwrites or removes a file names each, a file edited by hand since as `changed by hand`, and asks once, yes or no; off a terminal, and always with `--json`, it refuses without `--yes`, exit 2. It removes only a file it wrote, and of a file it did not write reads only one at a path it renders, to name it before it overwrites it. It reaches each path from the project's folder one name at a time and never follows a link: a link on a folder of a run folder's path, or on `.spwn/applied.json`, is refused by plan and apply alike, naming it, exit 2, and nothing is written. The declaration is never read through a link either: `project`, `plan`, `drift` and `apply` refuse a link in `spwn.yaml`, `spwn.lock` or under `spwn/`, and a name there that is neither a file nor a folder, a FIFO say, naming it, exit 2, before any of it is read; the project's folder itself may be a link. Nor does the declaration direct a read outside the project's folder: an agent that is not one folder name of `spwn/agents/`, a `knowledge:` path outside the project and an `@`-import outside it are each an error of the plan that names its field and says it leaves the project, whether that path is there or not, so apply refuses it, exit 2, and nothing outside is opened. A folder or a FIFO at a rendered file's name is an error of the plan, which apply never takes away; a file where a run folder or one of its folders goes is refused by plan and apply alike, naming it, exit 2, and never removed. A change it cannot make, a folder it may not write in say, stops it there, exit 2, naming the path; it records what it wrote before, so the next apply takes up where it stopped, and says so when the record itself could not be written. A second apply writes nothing and says `Nothing to apply.` It builds no image and starts no agent.

### `station teams`

```sh
station teams [<team>] [--json] # the teams declared for you by slug, with the agents of this folder's project that name each; one team's detail
```

A team is a file, `~/.spwn/teams/<team>.yaml`, declared for every project of the user, with a name, a description and the color it names (a hex or a name, shown as written); an agent belongs to it by the `team` of its `agent.yaml`, which `station agents edit <agent> --team <team>` writes. No command writes a team. A slug no team file has is refused, exit 2. With `--on` a VM of this Mac, it reads that VM's teams and the agents of the project's guest folder, refused like `station agents` when no share reaches the project. `--json` is `station.teams.list/v1`, or `station.teams.show/v1` for one team.

### `station automations`

```sh
station automations [<automation>] [--json] # the automations this folder's project declares, with when each last ran and how often; one automation's detail
station automations log [<automation>] [--last <n>] [--json] # the runs written for the project, oldest first; one automation's when it is named
```

An automation is declared in the project's `spwn.yaml`, bound to one of its agents by a cron schedule or a watched folder. These commands only read: they list the declarations and the runs recorded in `.spwn/runs.jsonl`, and schedule, fire and enable nothing, since Spwn runs an automation only while it is open and that is not built yet. A folder with no project, or a project with no automation, lists none. A name the project does not declare is refused, exit 2. A manifest that does not read is refused the same way (`automations.invalid-project`, with the manifest's own error and the file to open) by the list, the detail and the log of a named automation; `automations log` with no name reads no manifest and prints the runs of the file anyway. Two groups of the manifest may declare one name: the list then shows each as `group/name`, `automations <name>` and `automations log <name>` refuse the bare name (exit 2, naming the groups), and `group/name` picks one, in the detail and in the log alike. An automation named `log` is listed with a finding and has no bare detail, `automations log` being the command; `<group>/log` shows it and its runs are `automations log <group>/log`. A run shows what fired it (the trigger and its reason: on-time, catchup, create:<file>), how long it took, the catch-up slots it collapsed (`missed`) and the paths a file burst coalesced (`paths`); the detail of an automation says whether its last run passed (`last_ok` in `--json`, null when none ran). `--last` keeps the newest runs, 50 when not given, and `--last 0` keeps every run the file holds. The log says when it shows fewer runs than the file holds (`50 of 73: --last 0 shows all`) and how many lines of the file it could not read (`1 line of .spwn/runs.jsonl could not be read`); `--json` carries both as `total` and `unreadable`. With `--on` a VM of this Mac, they read that VM's project at the guest folder, refused like `station agents` when no share reaches it. `--json` is `station.automations.list/v1`, `station.automations.show/v1` or `station.automations.log/v1`.

## Spwn

These commands act on Spwn itself, or read across what it did: this machine at a glance, its setup, the machine a terminal uses, the journal and the Machinefiles.

### `station overview`

```sh
station overview [--all] [--json]
```

This machine at a glance (SPEC §5.1): a first line that sums it up (`This Mac · macbook-pro · 5 things need you · Engine running`; a VM's names its kind and how long it has run), then Needs you, every section's findings ranked broken first and then by the one section order, five of them unless `--all`; then one line per section the machine has, in that order (a Mac: Activity, Apps, Settings, Disk, Ports, Accounts, Background, Awake, then its VMs; a workspace: the sections it has; a server: Activity, Ports and SSH, and `Apps, settings, accounts: not managed by station`); Activity's line is CPU, memory, free space and network in, from the engine's history when it spans a rate, else one sample now, and its findings join Needs you; then one or two `→` next steps. Each line is that section's own read, so its counts equal what the section's own command prints. A fix that is a call prints as the command that runs it: the input it acts on is its word, each other input a flag, `--confirm=<name>` answers a question it asks typed, and `--on <machine>` names where it runs when that is not the machine reading (`station machines restart infra --confirm=infra --on mac-mini`). `station overview --on <vm>` and `station machines <vm>` print the same bytes: `On <vm>:`, the VM's first line from its Mac (`linux-1 · Workspace on This Mac · Ephemeral · up 2 h`), the sections it reads inside it, then its Health; a VM that does not run, or whose guest does not answer, shows its first line and Health alone. `--json` prints `station.overview.read/v1`: `machine`, `findings[]` (each `fix` one of `call`, `command`, `path`) and `sections[]` with their counts. On a terminal, `overview`, `apps` and `disk` draw every row at once and fill each as its read answers ([instant reads](../concepts/design-system.md#instant-reads)); piped or with `--json` they print one complete document.

### `station init`

```sh
station init [--name <name>] [--machinefile <name|path>]
station --version
station help [command]
```

`init` makes this machine known to Machines, safe to run again: it names this Mac (asking the name on a terminal, taking `--name` otherwise; off a terminal, a Mac with no name yet is refused, exit 2, `Hint: station init --name <name>`), writes the one managed block of `~/.zshrc`, writes the apps' own files (`relocate`), then moves what the apps left at the home root where those files point ([Disk](disk.md#relocation)). Before it moves anything it prints `# To move`, one row per move (`~/.bun/bin  to ~/.local/share/bun/bin  52 MB`, or a file it deletes, `deleted: its app writes it again`), then moves them with no question, on a terminal or off one: a moved file keeps its data, and a deleted one is written again by its app ([ADR-097](../decisions/097-a-confirmation-follows-its-consequence.md)). It installs no app, runs no package manager and asks no password; the engine comes with the first VM command on a Mac with tart, or `station engine start`. On a terminal it opens on a line that sums it up and ends on the empty state of what the machine carries (`Machinefile  none bound by init: init --machinefile binds one`), then at most two next steps: `exec zsh` when it wrote the block, `station init --machinefile <name>`, `station apps install <app>`. `--machinefile` then binds this machine to that Machinefile and runs `apply` on it: init's page names it (`Machinefile  downloads/laptop.machine.yaml: applied below, which binds it`) and drops the step that says how to bind one, then come its plan and the run. Inside a workspace it writes the block and the layout only; inside a server it sets nothing up, the server's own tooling keeps it. `--version` prints `station <version> (<commit>)`. See [Getting started](../start/getting-started.md) and [Dotfiles](dotfiles.md).

### `station use`

```sh
station use <machine> # every later command of this terminal runs there, as --on would
station use - # back to the machine used before
station use # the machine this terminal uses, and since when
```

`--on` still wins for its one command; other terminals stay where they are. See [Machines](machines.md#this-terminals-machine).

### `station journal`

```sh
station journal [--json] # what was done on every Mac and its VMs, by day
station journal --on <mac> [--json] # that Mac and its VMs alone
```

The last seven days of every Mac's journal, newest first: a section per day, a night's upgrade apart from its day with what it moved summed, each line with its time, the machines it concerns and who did it; then the Macs not heard from, each with why. With `--on`, one Mac's lines and its VMs', unsummed. `--json` prints `station.journal.read/v1`. A VM keeps no journal, so it is refused there. See [Journal](journal.md).

### `station machinefile`

```sh
station machinefile [--json] # the Machinefiles this Mac reads, and whether their machines match them
station machinefile plan [--machinefile <name|path>] [--phase image|boot|apply] [--diff] [--json]
station machinefile apply [--machinefile <name|path>] [--phase image|boot|apply] [--remove] [--yes]
station machinefile drift [--machinefile <name|path>] [--json]
station machinefile update [--machinefile <name>] [--json]
station machinefile explain [<key>] [--machinefile <name>] [--json]
```

Bare `machinefile` is `machinefile.files` of this Mac: one section per shelf (Yours, Mixins, Shipped), one row per Machinefile with what it makes (`workspace · ubuntu`, or `mixin`), where it is read (`user`, `shared`, `built-in`) and how the machines that follow it stand with it (`1 machine differs.`, `Every machine matches.`, `No machine follows it.`), the page's verdict under the first section. It changes nothing; the files are the ones this Mac reads, so `--on` is refused, exit 2. `--json` prints `station.machinefile.files/v1`: the page's `verdict` and `tone`, and each file's `name`, `shelf`, `origin`, `path`, `kind` (`mixin` for a mixin), `os`, `followers` (`machine`, `mac`, `state`), `verdict` and `tone`.

`plan` resolves the machine's Machinefile (the one named, else the one it is bound to, else the one named after it) and reads each package through its family: it opens with what the run asks of you, once per need (`◐ homebrew  asks your password once`, where this machine's sudo asks one), then what it installs and sets, what is adopted (listed, and present by another way, or by another of its methods, which it names: a state under `Adopted`, never installed again and never work) and what it leaves as it is (a setting you changed by hand, a read that did not answer), what is extra (installed by Machines for it, listed no longer) and what has to happen once applied, each once (`the Dock restarts`). It changes nothing. A package that declares `shell` adds a row of its own under what it sets, `~/.zshrc [station:postgres/path]: 1 line` (`Machines' earlier text, written anew` when the block is still what Machines last wrote and Machines writes it otherwise now), and one `Left as it is` when you edited the block by hand; a block Machines wrote for an extra is extra too, with what `apply --remove` does with it: `written by station, goes with postgres` when the app goes, `kept with postgres` while it stays, `edited by hand since station wrote it: kept, and yours` ([Dotfiles](dotfiles.md#package-blocks)). `--diff` replaces the table by the diff of each file apply would write, from what the file holds (`-`) to what apply writes (`+`): each shell block of `~/.zshrc`, and each setting's file (`station settings diff` drawn the other way); it says the table when there is none. `--json` prints `station.machinefile.plan/v2`, its `blocks` the shell blocks (`package`, `id`, `condition`, `action`); `apply --json`'s `station.machinefile.apply/v2` lists the blocks the run judged, each with its `result` and `detail`, so an extra's block is there with `--remove` alone.

`apply` prints the plan, then carries it out one package at a time, by `apps install` and `settings apply`, and prints the run: each package with its mark, version and method. A failure fails that package, exit 1, and each package that waits on it is `not run · requires <package>, which did not install`; every other package goes on. An interruption marks the package it stopped at `interrupted (Ctrl-C) · station machinefile apply resumes` and each one after it `not run`, exit 130 (a package that finished just before is kept as its install answered; one whose install handed nothing back is in the ledger when a fresh read of the apps gives its row within 15 seconds, unrecorded otherwise, so never removed), and the next `station machinefile apply` finishes from there. Off a terminal, a plan that asks you anything (a password, a license, a question; with `--remove`, what uninstalling an extra asks) is refused before any effect, exit 2, `Hint: run station machinefile apply on a terminal`. A password is asked only where this machine's sudo asks one: a plan that holds a sudo need reads it once, by `sudo -k -n true`, which never prompts, runs nothing but `true` and is given 5 seconds, a read that cannot tell counting as asking; a VM Machines made asks none, so `apply` inside one installs Homebrew off a terminal. `--machinefile <name|path>` binds the machine to it first; a file named by its path is copied into `~/.config/station/machinefiles/` under its name, the same words copying nothing, and one whose name that folder already holds in other words is refused before any effect, exit 2: Machines never replaces your Machinefile. `--remove` also removes what the ledger says Machines installed for this Machinefile that it lists no longer, and only while it reads there by the method that installed it, dependents first, asking on a terminal (`--yes` off one, else exit 2); an extra another app there still needs (it requires it, provides it, or manages it, by that app's method as `station apps` shows it; a setting's requirement keeps no app), whoever installed that app, is `required by <app>: kept`, with its record, and so is one that an extra Machines cannot uninstall, or whose uninstall failed in this run, still needs; an extra Machines cannot uninstall (a script's install that names no `undo`, an apt app Machines recorded adding nothing of) is never taken back and asks nothing, `station cannot uninstall it: kept, and yours: remove it by hand if you want` (`required by <app>: kept`, with its record, when an app there needs it); an extra already gone, or held by another method since, or that station cannot uninstall and nothing needs, is said and kept, and the ledger lets it go; one whose manager is missing here, even one that reads absent, is `its manager <manager> is missing: kept`, with its record; a manager that holds packages of yours (what its methods list that no app of the catalog installs, `npm i -g typescript` for node), or whose list does not read, is `holds <n> packages of yours: kept`, with its record, and taken back once it holds none. A `vault:` value is shown as itself and never printed.

`drift` is the plan read as conditions, one row per package: `present`, `absent`, `outdated`, `changed`, `not managed`, `extra`, `unknown`. It never acts. `--on <vm>` judges the packages on the VM and reads its ports and accounts on its Mac: each one held otherwise than its file gives it is a row of `Ports and accounts` (`port 3000  changed · shared public, its file opens it to private`). `--json` prints `station.machinefile.drift/v3`.

`--phase` plans or carries out one phase of the Machinefile: `image`, what a VM image bakes (`vm.image.bake` and what each baked package requires), the one `station images build` runs inside its build VM; `boot`, a VM's first start; `apply`, the default, every package but a boot or image package's, the baked ones included. A phase none of these is refused before any effect, exit 2, and so is `--remove` with another phase than `apply`, since another phase carries only a part of the Machinefile ([Machinefiles](machinefiles.md#phases)).

`apply` and `update` write the Machinefile's lock, `~/.config/station/machinefiles/<name>.lock`, once their run ends, interrupted or not ([Machinefiles](machinefiles.md#the-lock)). `apply` installs an app the lock holds by the lock's method, at its version where that method pins one, and the plan says it (`○ python  mise  locked 3.12.7`). `apply` reads the shared source where its clone stands, and `update` moves it forward to its default branch's newest commit; a lock that does not decode is refused at its file, `Hint: remove it: the next station machinefile apply writes it anew from what is there`.

`update` reads its plan where the shared source's clone stands, refuses before any effect what it cannot run here, then moves the source forward to its default branch's newest commit and reads the plan there, going on from the clone as it stood when the source does not move (offline, a rewritten branch, a commit that does not resolve: a `Source` section says why); a clone whose own commit no longer resolves is moved first and read at the newest commit alone, so a fix pushed upstream is taken; then it moves every app the Machinefile lists that reads there by one of its methods as far as its `version` lets it ([Versions](machinefiles.md#versions)), by that method, the managers first (homebrew's index by `brew update`, mise, node) and then the rest, each in the plan's order, then writes the lock anew and prints what moved in it (`gh  2.98.0 by brew, was 2.97.0 by brew`, the shared source's commit). An app of mise's or bun's is installed again at its range (`within 22`, `held at 22.11.0`), one exact version of apt's or a release's is kept, `pinned 2.43.0 by <machinefile>: kept`, and a newer version past a range is said waiting, with the line that takes it (`node  25.1.0 available: change '22' to '25' in <file>`); a copy no method holds is yours and left as it is, and an app not there is apply's, except one the lock holds: a locked version that can no longer install is no dead end, so update installs that app by the lock's method at the newest the Machinefile allows and the lock follows (`ripgrep  1.0.0 by brew, was 14.0.0 by brew`). Off a terminal, an update whose updates ask you anything is refused before any effect, exit 2, `Hint: run station machinefile update on a terminal`; an interruption marks the app it stopped at and each one after it, exit 130, `station machinefile update resumes`. It updates neither Machines itself (re-run the install, [Release](../guides/release.md)) nor what the Machinefile does not list (`station apps update`). `--json` prints `station.machinefile.upgrade/v2`: each `line` as it ended, the `lock`'s moves (`app`, `was`, `now`), the `source` when it moved, each version `waiting` (`app`, `newest`, `range`, `moved`, `file`), `interrupted`, and `unpulled`, why the shared source did not move, when it did not. One apply, update or night runs on a machine at a time: a run that finds another holding it waits a few seconds, then is refused, exit 2, `Hint: run it again once that one ends`; one interrupted while it waits exits 130.

`explain` resolves the Machinefile this machine is bound to (else the one named after it), or `--machinefile <name>`, against the catalog, and prints each value with its origin (`user`, `shared`, `built-in`, `overlay`, `this machine`, `lock`) and the Machinefile or package that says it; a key (`vm`, `apps.jq`) explains itself and every key under it. Every refusal of the resolution exits 2 with its own words: no Machinefile of that name, a file refused at its line, a newer schema (`update station`), `vm` on a computer, a cycle, a package with no method for the system or a range its method cannot hold, a package the catalog does not hold, a value that is none of the package's options, a key it says nothing of; a mixin (a Machinefile that says no `machine:`) answers the values its own chain says, with no step judged against a machine; a shared source that cannot be read exits 1. `--json` prints `station.machinefile.explain/v1`. The format is [Machinefiles](machinefiles.md).

## Plumbing

These commands exist for the machine's own use. They are hidden from help and the owner never types them.

| Command                       | What runs it and what it does                                                                                                                                                                                                                                                                                                  |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `station shell zsh`           | the `~/.zshrc` block evaluates it at every shell start: PATH, the mise activation, Machines' completion, and in a VM the [profiles'](ports.md#what-a-package-sets) exports. Output on stdout only; another shell or none is refused                                                                                            |
| `station shell prompt`        | the prompt's `custom.station` segment runs it: `on linux-1` while this terminal [uses](machines.md#this-terminals-machine) another machine, nothing otherwise; it never counts as a use                                                                                                                                        |
| `station completion <shell>`  | cobra's completion script, for a shell the block does not serve                                                                                                                                                                                                                                                                |
| `station engine run`          | the launchd agent `com.jterrazz.station.engine` runs it on a Mac, `com.jterrazz.spwn.engine` with Spwn.app: the VMs and their ports, and the ssh key agent on `~/.local/state/station/agent.sock`, which serves this Mac's device key and relays any other key to the session's own agent ([Machines](machines.md#the-engine)) |
| `station guest serve --stdio` | the Mac's engine runs it inside each VM over its one guest connection: the VM's calls, and what listens there ([The API](api.md#the-transport))                                                                                                                                                                                |
| `station engine attach`       | another Mac runs it over ssh: bridges ssh's stdio to this Mac's engine socket ([The API](api.md#the-transport))                                                                                                                                                                                                                |
| `station machines pipe <vm>`  | a VM's `~/.ssh/config` ProxyCommand runs it: ssh's bytes to the VM's ip behind its Mac ([Machines](machines.md#ssh))                                                                                                                                                                                                           |

`--on` is refused on `shell`, `completion`, `help`, `use`, `engine run`, `engine attach`, `guest` and `machines pipe`: none of them sends a call.

## Exit codes

| Code | Means                                                                                                                 |
| ---- | --------------------------------------------------------------------------------------------------------------------- |
| 0    | Done, or a read that found findings                                                                                   |
| 1    | `failed`, `unreachable`, `engine-stopped`, `far-too-old`, `declined`: an operation failed, or you said no             |
| 2    | `refused`, `not-found`, `not-for-this-machine`, `needs-confirmation`, `needs-terminal`: before any question or effect |
| 130  | `interrupted`; nothing is printed                                                                                     |

The exit code is computed once, from the error's type ([Architecture](../concepts/architecture.md#the-error-model)). A command over several packages prints every outcome, then exits 1 with `<n> of <m> <noun> failed` if any failed.

## Related

- [Design system](../concepts/design-system.md): the forms a document takes.
- [The API](api.md): the calls behind every command.
