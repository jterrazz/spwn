# Machines

All machines are every machine the owner runs: the physical Macs and the VMs on each. One list shows them, any Mac can drive any other, and a VM is made, run and deleted from the Mac it lives on.

| Section           | Answers                                                      |
| ----------------- | ------------------------------------------------------------ |
| Macs and VMs      | What a machine is, and how it is named and reached           |
| Lifetimes         | When a VM ends: poweroff, reboot, Mac reboot, `delete`       |
| The commands      | Create, start, stop, restart, delete, unlock                 |
| `--on`            | Running a command on another machine                         |
| `station use`     | The machine this terminal's commands run on                  |
| Tailscale         | The Tailscale app on each Mac, read only                     |
| SSH and Herdr     | How one name reaches a Mac, a VM of This Mac, of another Mac |
| The engine        | The one process per Mac that runs the VMs                    |
| Round 5 migration | What setup retires, and what the owner removes by hand       |
| Mac readiness     | What a Mac that hosts servers must have, each item measured  |

## Macs and VMs

A Mac registers itself with `station setup --name <name>`; another Mac is added by its Tailscale name with `station machines add <mac> [--user <login>] [--key <file>]`, once its ssh and its station answer, and forgotten with `forget`, which never reaches it. A VM is cloned from an [image](images.md) on a chosen Mac. A machine's kind is `mac`, `server` or `workspace` ([ADR-017](../decisions/017-servers-and-workspaces.md)); its system, `macOS` or `Linux`, is its own word. A VM learns its kind from the file its Mac wrote at create; a macOS VM is told from a Mac by `kern.hv_vmm_present`. A VM refuses the commands that only Macs do, with a hint to run it on its Mac.

A machine has one name, used for `preferences.json`, the `~/.ssh/config` block, Herdr and `--on`; a Mac's name is also its tailnet node. A VM is never on the tailnet: it is reached through its Mac. Every VM has the one login `vm_user`; a Mac's login is its own `--user`.

`station machines` is All machines: a first line that counts the machines, the running, the stopped and what needs you; each Mac with its VMs under it, in the columns MACHINE, KIND, STATE, CPU and MEMORY; then Needs you. A Mac that does not answer in 5 seconds keeps its row, `✗ Unreachable`, and is a finding, never a failure; a Mac that runs another station says which and where to update it. Piped, each machine is one record (glyph, name, kind, its Mac, state, CPU, memory); `--json` is `station.machines.list/v1`.

Each Mac says what it is, read by sysctl and never by a subprocess: its model identifier (`hw.model`), the name that model is shown by (from the catalogue's table, `Mac` for one it does not hold), its chip (`machdep.cpu.brand_string`), its memory and cores, and whether it has a lid (a laptop). A Mac that falls silent keeps what it last said it was. `machines --json` carries it as each Mac's `hardware`. Its capacity is its memory, the 4 GB macOS keeps, what each VM reserves and what is left free for a new VM: a VM reserves its memory while it runs, and a server its record wants running keeps its share while it is down, so the free figure never jumps when it falls and comes back; a create asking more than is free is refused. `machines --json` carries it as each Mac's `capacity`, and names as `roomiest` the Mac that answers with the most free memory (This Mac on a tie), where a new VM goes when none is named.

A VM being created is on the list from its create's first frame, `Creating`, its memory reserved at once: its record carries the step its create runs (`cloning the image`, `reserving its ports`, `booting`, `waiting for an address`, `waiting for SSH`) and how far it is, per mille, until the create ends, made or kept failing. A create that has not moved for 30 minutes died with what ran it: its VM reads stopped, `was left half made`, to start or delete. Once made, its engine tells `vm_ready` (`linux-4 is ready on mac-mini`) and the VM is `fresh` for 8 s. `machines --json` carries `birth` (`progress`, `step`) while it is being made, and `fresh`.

A Mac that falls silent is shown as it last answered: for 30 s since its last answer it is fresh, then `stale`, as is a Mac never heard from. A laptop that last answered on battery with its lid closed is `asleep` when it falls silent, told apart from one that is `unreachable` for a reason station cannot tell. `machines --json` carries each Mac's `reach` (`answers`, `asleep`, `unreachable`) and `stale`.

```text
5 machines · 4 running · 1 stopped · nothing needs you
 MACHINE KIND STATE CPU MEMORY
 ● This Mac Mac Engine running
 ├─ ○ api Workspace Stopped · permanent 4 CPU 8 GB
 └─ ● linux-1 Workspace Running · throwaway 4 CPU 8 GB
 ● mac-mini Mac Engine running
 └─ ● infra Server Running · always on 8 CPU 24 GB
```

`station machines <machine>` is that machine's Overview, the one `station overview --on <machine>` prints.

A server has four sections: Overview, Activity, Ports and SSH (SPEC §3.2). Its Overview adds `Own Tailscale device` when its tooling put one on the tailnet (station never installs nor blocks it) and its ssh, for debugging; its SSH section is the one door, the path that reaches it and what station leaves to the owner's tooling. Apps, Settings, Disk, Accounts, Background and Awake are refused on it, exit 2: `infra is a server: station does not manage its apps`, `Hint: manage it with your own tooling: ssh infra`. `apps update --all-machines` and `accounts --all-machines` skip a server by its kind and say so.

## Lifetimes

A workspace is throwaway unless it is made `--permanent` ([Glossary](../concepts/glossary.md)); a server is always on (its record's lifetime is `always-on`, the one mark of its kind). No workspace restarts by itself (SP-4), and a stop never destroys a VM (SP-5). A server whose wanted state is Running starts with its Mac's engine; a stop sets its wanted state to Stopped, so it stays stopped across its Mac's reboots until `start` sets it back to Running. A Mac's servers are listed before its workspaces:

| Event                          | Throwaway workspace                                   | Permanent workspace                             | Server                                                         |
| ------------------------------ | ----------------------------------------------------- | ----------------------------------------------- | -------------------------------------------------------------- |
| `poweroff` inside the VM       | deleted: disk, reserved ports, ssh blocks, record     | stays stopped until `start`                     | stays stopped, wanted Stopped, `powered off from inside`, told |
| `reboot` inside the VM         | comes back; never deleted                             | comes back                                      | comes back at once, counted, no wait, not told                 |
| Mac reboot or power cut        | stopped, `ended with its Mac`                         | stopped, `ended with its Mac`                   | starts with its Mac's engine, once its disk is unlocked        |
| a crash, or an end not read    | kept, `ended unexpectedly · start or delete it`       | kept, `ended unexpectedly · start or delete it` | boots again after its wait, counted, told                      |
| `station machines stop <vm>`   | stopped, kept with its disk until `start` or `delete` | stopped, kept with its disk until `start`       | stopped, wanted Stopped until `start`                          |
| `station machines delete <vm>` | deleted                                               | deleted                                         | deleted                                                        |

The VM tells its Mac when it powers off: a unit in the image writes `station-intent=poweroff` to its console, and [the engine](#the-engine) reads it when the VM's run ends. The Mac cannot otherwise tell a reboot from a poweroff, so any end it cannot read as a poweroff never deletes a throwaway workspace.

### Keeping a server

The engine keeps a server running and healthy. A run that crashes, or ends in a way it cannot read, boots again after a wait that doubles from 5 s: 5 s, 10 s, 20 s, 40 s, then 60 s each time. A run that stays up 10 min brings the next wait back to 5 s; the count of restarts goes on (`Restarts ↺ 3 · last just now`). Five restarts within 10 min are a loop: the server reads `✗ In a loop · ↺ 5 in 10 min · waiting 60 s`, the owner is told once, and the engine keeps trying every 60 s. A reboot from inside is counted with no wait, and nobody is told. A poweroff from inside is the server's own word: it stays stopped, wanted Stopped, until `station machines start`.

Its check (`--check`, a URL that answers 2xx or 3xx, or a `host:port` that accepts a connection) is run from its Mac. A health check that never passed since the boot is waiting (`◐ Check waiting`), not a failure; one that passed then fails is `◐ check failing since 10:01`, a finding under Needs you. A failing check never restarts the server: what runs inside is its tooling's. When it passes again the finding goes and the owner is told `infra is back · all 4 health steps pass`. A restart, a loop, a poweroff from inside and a return are each told once, to the Mac's notifications and to every watch of the engine.

While a server is healthy its check runs every 60 s, and its engine keeps how long each run took, in its memory alone (the last 60: the latency now, the median and the peak with its time). A restart walks the four health steps, Running, Has an address, SSH answers and Check passes, and the engine shows the step it runs and when it began (`2 of 4`, Has an address) until all four pass. Each server's last 30 days live in `~/.local/state/station/uptime.json`, written by its Mac's engine alone: a day before the record started is `none` (nothing is back-filled), a day an incident still held at its end, or holds now, is `down`, a day whose downtime all recovered is `restart` (so today turns `restart` the moment the server is back), any other `ok`; with the last 20 incidents, from, to and cause, and the availability over those days to two decimals, rounded down. A server goes down on its record when its run ends unexpectedly, it powers off from inside, or its check fails since it passed; it is back once all four steps pass. Why it went down is read from the last 40 lines of each of its run's two outputs, its serial console then `tart run`'s own, blank and timestamp-only lines skipped: an `Out of memory: Killed process` line is `out of memory`, a `Kernel panic` line `kernel panic`; with none, a poweroff from inside is `powered off from inside`, a crash `the hypervisor stopped it`, a failed check `its check failed`, any other end `it stopped`. Its record keeps that cause and three of those lines, the cause's among them, until it is back. `station machines log <vm>` prints every line its runs wrote, read on its own Mac (`--on`). `machines --json` carries each server's `uptime`, `latency`, while it restarts `restart`, and while it is down `cause` and `last_lines`. Requests per minute are not observable by station: the card shows the latency alone (owner decision D1, pending).

## The commands

```sh
station machines create api --permanent --memory 12G --disk 80G
station machines create infra --server --check https://infra:6443/readyz --on mac-mini # always on; nothing set up inside
station machines start api # a stopped workspace, its disk as it was left
station machines stop api # asks yes or no; kept with its disk
station machines stop infra --on mac-mini # a server: asks its name typed (--confirm=infra), --yes answers nothing
station machines edit api --memory 16G # applies when api next starts; a disk only grows
station machines edit infra --check infra:6443 # a server's health check, read by its next health run; --no-check removes it
station machines ssh api # prints the ssh command: ssh admin@api
station machines restart mac-mini # a Mac: asks naming the VMs that pause, a planned restart that unlocks itself, waits for its station, starts those VMs again; a VM: power off, run
station machines delete api # destroys the VM, its leases and the secrets lent to it
station machines api # its Overview, as station overview --on api
station machines unlock mac-mini # after a power cut, on its last LAN address: you type its password into macOS's pre-boot prompt
```

`create` makes a throwaway workspace unless `--permanent` is given, and asks which Mac when two or more are reachable. Defaults are 4 CPUs, 4 GB, 20 GB and image `linux`, each size with its unit. A size above what the Mac can give is refused before any effect (`not enough memory on This Mac: 60 GB asked, 4 GB left for VMs`): macOS keeps 4 GB, and the running VMs hold theirs. It reserves the ports of every [profile](ports.md#profiles) before the VM first runs; a step that fails before that rolls everything back. It then prints the health steps as they pass (Running, Has an address, SSH answers), its ports, and `linux-2 Created` with its size. A server (`--server`) is named, always on and behind its Mac (`--permanent` and `--network bridged` are refused), sized 4 CPU · 8 GB · 50 GB unless asked; station reserves no profile port and runs no first boot inside it: after SSH answers, its check is `◐ Check waiting` (what answers it comes with your tooling), then `infra is a server`, `Station sets up nothing inside`. Made on a Mac that reports a battery, it adds `◐ This Mac is a laptop: box stops whenever it sleeps · see Mac readiness`. From the first run on the VM is kept whatever fails: `linux-2 did not boot: …`, or `linux-2 is kept, failing Has an address: none after 60 s`, each with start or delete as the fix. A running workspace whose guest connection fails is a finding, `fails SSH answers`. A Mac that does not answer keeps its row in `station machines`, `Unreachable · last answer 03:12`, and the VMs it held at that answer as `? Unknown`; `station machines <vm>` of such a VM reads `infra is on mac-mini, which does not answer`, exit 0. `--network bridged` is for macOS VMs only (an iPhone on the Wi-Fi reaching the VM); a Linux VM shares its [ports](ports.md) instead. With Tailscale HTTPS off, the profiles' variables hold the local URL and `create` adds a `◐ ports` row naming the admin step. `delete` asks the name typed (`--confirm=<name>` in a script, never `--yes`), and takes with the VM the ports declared public for it. With the engine stopped, start, stop, restart, edit and delete are refused (`Hint: station engine start`).

`create`, `start`, `stop`, `restart`, `edit` and `delete` need the [engine](#the-engine): with it down they refuse before any change and the VM is kept (`Error: the engine is stopped on This Mac`, `Hint: station engine start`; without the agent's plist, `is not installed on This Mac` and `station setup`).

## On another machine

`--on <machine>` runs the command's call there and renders its answer here: `station overview --on mac-mini`, `station apps install gh --on api`. Another Mac is reached through its engine: one `ssh -o BatchMode=yes <mac> station engine attach`, which speaks `station.api/v2` frames over ssh's stdio; a VM of this Mac through this Mac's engine, which runs `station guest serve --stdio` over its one connection to the VM. A VM of another Mac rides that Mac's engine: its name is looked up in this Mac's own records first, then asked of every other Mac at once (`machines.list`, 4 at a time, 5 s each); station never runs `ssh <vm>`. A name no Mac holds is `Error: no machine named "<name>"`, `Hint: station machines`, exit 2, with nothing dialed under that name; while a Mac is silent the VM may be its own, so the answer is that Mac's `does not answer`. The far side returns data, so the output is this binary's, in tty, pipe and JSON alike. Each connection opens with `Hello`, the far binary's version and the calls it serves: a call it lacks is `Error: <mac> runs station <version>, which has no <call>`, `Hint: update station on <mac> (scripts/install.sh there)`, exit 1. A connection to a Mac serves every call while it answers, and nothing outlives it: a Mac asleep at one call, or whose link dropped, is dialed again at the next, and a call the open connection's `Hello` lacks is asked again of a fresh one, so a Mac upgraded meanwhile serves it. A far refusal is worded by this binary's table; a code this binary does not know keeps the far side's own words. A machine that does not answer is a finding for a read, `✗ <machine> does not answer`, exit 0, and for an action `Error: <machine> does not answer`, `Hint: station machines <machine>`, exit 1; a VM of such a Mac is `<vm> is on <mac>, which does not answer`, worded the same way. It is refused on `shell`, `completion`, `help`, `man` and the hidden plumbing (`engine`, `engine`, `guest`, `machines pipe`, `machines supervise`). Bare `station --on api` opens the TUI here, its calls on that machine. After a remote create, delete, add or forget, the local ssh blocks refresh.

### This terminal's machine

`station use linux-1` says `On linux-1 in this terminal. Other terminals stay on This Mac.` and every later command of this terminal runs on linux-1, as with `--on linux-1`; `--on` still wins for its one command. `station use -` goes back to the machine used before (`Back on This Mac (was linux-1).`), `station use this-mac` back to This Mac, and bare `station use` prints the machine and since when. The choice is this terminal's alone: the shell block exports `STATION_TERMINAL`, an id made at each shell start, and the choice lives in `~/.local/state/station/use/<id>`; a choice no command used for 7 days is forgotten. A terminal without the block is refused, exit 2 (`Hint: use --on linux-1, or run station setup`). A used machine that is gone is said once by the next command (`linux-1 is gone (deleted at 14:10) · back on This Mac`, the time when this Mac's station deleted it), which runs on This Mac and clears the choice. `shell`, `completion`, `help` and the plumbing run here whatever the choice.

The All machines list asks each other Mac's engine for its own view (`machines.list`), 4 at a time, 5 s each; a Mac that does not answer is a row of its own, never the others' delay. On a terminal the rows fill as each Mac answers; piped and in JSON the list prints once, whole.

## Tailscale

Each Mac runs the Tailscale app; the owner logs in through the app. Station only reads it: the first line of `station ports` shows the node, its MagicDNS name, the CLI it ran and what `tailscale serve` exposes, and station drives nothing but `tailscale serve` for [shared ports](ports.md#serve-and-https) and `tailscale cert`, which [warms the certificate](ports.md#serve-and-https). A VM has no node, no key and no tag; the owner's Tailscale policy needs no line for station.

The app comes in two variants, and station takes either:

| Variant      | Bundle id                 | How it got there                                                                                                                    | Its CLI                                                            |
| ------------ | ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| `standalone` | `io.tailscale.ipn.macsys` | the cask `tailscale-app`, `station apps install tailscale`                                                                          | `/usr/local/bin/tailscale`, which the cask's pkg installs          |
| `app-store`  | `io.tailscale.ipn.macos`  | the App Store (`Contents/_MASReceipt`); `station apps` lists it `ok`, unmanaged, `app-store`, never upgraded nor removed by station | `/Applications/Tailscale.app/Contents/MacOS/Tailscale`, on no PATH |

Every `tailscale` call, the engine's included, runs the CLI resolved at that call by absolute path: `/usr/local/bin/tailscale`, then the App Store bundle's executable, then `tailscale` on PATH (launchd's PATH holds neither `/opt/homebrew/bin` nor `~/.local/bin`); none is the tailnet `absent`. Every call sets `SHLVL`: the App Store bundle's executable answers as a CLI only when `TERM`, `SHLVL` or `TERM_PROGRAM` is set, and launchd sets none of them. A `Warning: client version ...` line, a CLI older or newer than its engine, is dropped before the answer is read. `station tailnet`, `station status`'s tailnet row and `status --json`'s `tailnet` say the `variant` and the `cli` path used.

With the App Store app, `station setup`'s `apps` step makes `tailscale` answer in a terminal, an IDE and `studio doctor` by writing `~/.local/bin/tailscale`, two lines, `#!/bin/sh` and `exec /Applications/Tailscale.app/Contents/MacOS/Tailscale "$@"`: a symlink to that executable crashes (`bundleIdentifier is unknown to the registry`), so a symlink found there is replaced, the script's exact bytes are adopted, and a file station did not write, or another `tailscale` on PATH, is left alone. Until it is written `status` shows `◐ setup tools`. The same step uninstalls the Homebrew formula `tailscale` (an older CLI the app supersedes) once the app is installed: `● tools tailscale formula uninstalled: the app replaces it`; the formula alone stays, `◐ tailscale formula installed, app absent`.

## SSH and Herdr

Station writes one managed block per machine in `~/.ssh/config` on every Mac, refreshed by a bare `machines` and after every change, so `ssh api` works on every machine. Every block pins its host key in `~/.local/state/station/known_hosts/<name>` (`StrictHostKeyChecking accept-new`). The three shapes:

```text
Host mac-mini # a host: its sshd, reached by its MagicDNS name
 HostName mac-mini
 User jterrazz.agent
 IdentityFile ~/.ssh/id_ed25519

Host linux-1 # a VM of this host: no hop
 ProxyCommand /Users/jterrazz/.local/bin/station machines pipe linux-1

Host api # a VM of mac-mini, seen from macbook: one hop
 ProxyCommand ssh mac-mini -- "sh -c 'PATH=\"\$HOME/.local/bin:/usr/local/bin:\$PATH\" exec station \"\$@\"' station machines pipe api"
```

`station machines pipe <vm>` (hidden plumbing) carries ssh's bytes to the VM's current ip behind its own Mac; it never needs the engine. A VM trusts its Mac's key and every line of the Mac's own `~/.ssh/authorized_keys` through a managed block of its `~/.ssh/authorized_keys` (`# >>> station hosts >>>`), written by the guest, so whoever may log into a Mac may reach its VMs. Herdr drives that same ssh: `hc machine add api` (add `--temp` for a throwaway workspace). `station machines ssh <machine>` prints the command, never runs it, so `$(station machines ssh api)` does.

## Files and editors

Nothing of a Mac is ever shared into a VM: `tart run` gets no `--dir`, and no flag or setting adds one, because a mounted Mac folder would be a hole in the isolation. The ssh name is the one door, for files and editors alike:

```sh
scp report.zip api: # one file in
rsync -a project/ api:project/ # a folder, again and again
scp api:out.log . # and back
```

An editor on a Mac opens the VM by its Remote SSH to the same name: Zed, VS Code or Cursor on `ssh://api/~/code`. So there is no `cp` verb and no shared exchange folder.

## The engine

One engine per Mac, under the launchd agent `com.jterrazz.station.engine` that `station setup` installs once tart is there (`○ engine not needed until tart is installed` before). It runs every VM of the Mac detached (an engine restart never stops a VM), reads how each run ended (the rules of [Lifetimes](#lifetimes)), adopts the running VMs again at its start (its socket appears only once that and the serve reconciliation are done), keeps one ssh connection per running VM to its [guest](ports.md), and holds the reserved ports. It samples the activity of This Mac and of each running VM (from its guest; from its Mac's view of its process while the guest does not answer) every 60 s, every 2 s while a surface shows it (the menu bar open shows This Mac, and a machine's page open in it that machine; the menu bar closed shows none; the terminal UI shows the machine of its Activity or Overview, and All machines every machine of this Mac), and keeps it in its memory alone: the last 6 min every 2 s, the last 24 h every minute; a disk under 5 % free or another finding that breaks is told once. `machines create` returns once that guest is connected and has written the VM's keys block and its ports' env file (60 s, else the VM is kept, failing SSH answers). `machines start` and `restart` return once the new run's guest is connected too: a stop ends the guest's session with its run. That connection shares the VM's host key pin with `ssh`, whichever key type was pinned first. Why a guest does not connect is written to `~/.local/state/station/logs/engine.log`, one timestamped line per new reason. Its `tailscale` calls die with it: each leads its own process group, killed whole when the call ends or times out, and a call a killed engine left running is killed at the next engine start, from its record under `~/.local/state/station/children/`. The CLI is its client over `~/.local/state/station/engine.sock` (protocol `station.api/v2`, [The API](api.md)). With it down, `machines` and `ports` still read the files and show one engine row, and `status` the same row in its Machine section: `◐ engine not installed → station setup` when the agent's plist is absent, `◐ engine stopped → station engine start` when the plist is there but the job does not answer (another Mac's stopped engine is read and started through it, `station engine start --on <mac>`).

`station engine stop [--yes]` is Stop everything on this Mac: every running workspace stops with its disk kept, a throwaway one too (only `machines delete` destroys), all at once through the engine's `engine.stop-all`; the Awake session ends and the sharing of their ports with it; then the agent is booted out (`launchctl bootout gui/<uid>/com.jterrazz.station.engine`) and the stop is kept with who asked it, so `station engine` reads `stopped · since 13:40 by the menu bar`. A server is skipped and says so, and while one runs the engine keeps running for it: `infra skipped (server) · the engine keeps running for infra`; `station machines stop infra` stops it. A VM that will not stop keeps the engine running (exit 1). Off a terminal, a stop that would stop a VM needs `--yes`. `station engine start` bootstraps the agent again; the VMs it stopped stay stopped, and a login starts the engine again whatever stopped it.

## The v3 migration

`station setup` on a Mac station v2 set up moves, once, and says so: `~/.config/station/settings.json` to `preferences.json` (byte for byte, only while no `preferences.json` is there: `● preferences moved settings.json to preferences.json`), and the engine's job `com.jterrazz.station.daemon` to `com.jterrazz.station.engine` (`● engine running v3.1.0, its job moved to com.jterrazz.station.engine`). A second run says nothing of either. A `preferences.json` that does not read stops every command that needs it, at its line, and is never rewritten.

## Round 5 migration

`station setup` moves a Mac from the earlier rounds and says so, row by row:

- `layout`: the userspace tailscaled station ran is stopped (in the background when it carries the session) and its files removed; its node stays in the [admin console](https://login.tailscale.com/admin/machines) until the owner removes it. The three tailnet keychain accounts are deleted, and the row points at the [OAuth client](https://login.tailscale.com/admin/settings/oauth) to revoke. Every store is brought to its version: `preferences.json` loses any key no group names (the `tailnet` key among them); a record setup cannot read is left as it is, for the command that reads it to name, and a record a newer station wrote fails the step.
- `engine`: every per-VM agent `com.jterrazz.station.vm.<name>` is booted out and its plist removed, a VM keeping its wanted state, stopped or running; with any of those VMs running the step refuses (`Hint: stop the permanent VMs and delete the throwaway ones (station machines), then station setup`), never cutting a VM.

Left to the owner, by hand and whenever wanted: the old VM tag and its ACL lines, the OAuth client and each Mac's old userspace node in the admin console. The Homebrew formula `tailscale` is `setup`'s to uninstall once the app is there ([Tailscale](#tailscale)).

## Mac readiness

The Overview of a Mac that hosts a server, or that never sleeps (its `never-sleeps` setting applied), ends with Mac readiness, six items in this order, each measured on that Mac: restarts after a power cut (`restart-after-power-cut`), never sleeps (`never-sleeps`), logs in by itself (`autologin`), disk unlock after a power cut (FileVault on is `◐ needs you`, its fix `station machines unlock <mac>` from the same network), planned restarts unlock themselves (`fdesetup supportsauthrestart`, a fact), engine starts at login (the engine's job installed, else `station setup`). A setting not applied names its fix, and says `as root` where it needs it. A Mac hosts a server when one of its running VMs says it is one, asked of its guest's station. Any other Mac has no readiness, and reads no fact for it.

## Related

- [Ports](ports.md): what a VM listens on, and its URL.
- [Images](images.md): what a VM is cloned from.
- [Accounts](accounts.md): what reaches a VM, and what delete revokes.
- [Commands](commands.md): every flag.
