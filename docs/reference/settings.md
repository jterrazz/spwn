# Settings

`station settings` is what station sets up on a machine (dotfiles, logins, macOS defaults, shortcuts) and what it checks there without applying it. Each entry is a setting, named by what it sets; station's own choices are not settings, they are [Preferences](preferences.md).

| Section      | Answers                                                     |
| ------------ | ----------------------------------------------------------- |
| The packages | Every setting station applies, where it lives, what it does |
| Checked      | What station reads from macOS without applying it           |
| Values       | How a setting that needs values asks for them               |

## The packages

A setting that does not live on this machine's kind reads Not for this machine, and an apply of it is refused with exit 2. Each that lives here reads Applied (exactly as station applied it), Changed since (applied, then changed by someone: `station settings diff` shows how), Not applied (never applied, or still exactly as station applied it while station writes it otherwise now, so the next apply writes the new text without asking), or Off here (the owner's choice, kept as `off_here` in [`preferences.json`](preferences.md)). What station applied, its fingerprint and when, is kept under `~/.local/state/station/applied/`. The fingerprint is what the machine holds of the setting's effect, never what station would write: an edit by hand, or a part taken away (a file deleted, a workspace taken out of the multiplexer), moves it and reads Changed since; a part the setting gained that is not there yet (a new file, a new block, a new workspace) does not, so the setting reads Not applied and the next apply writes it.

| Package                     | Lives on   | Sets                                                                                                                                                                                                                                          | Reversible |
| --------------------------- | ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| `hushlogin`                 | all        | `~/.hushlogin`                                                                                                                                                                                                                                | no         |
| `tmux-dotfiles`             | all        | `~/.tmux.conf`, reloaded in a running tmux                                                                                                                                                                                                    | no         |
| `starship-dotfiles`         | all        | `~/.config/starship.toml`                                                                                                                                                                                                                     | no         |
| `github-login`              | all        | `gh auth login` over ssh; in a VM this token is the VM's own                                                                                                                                                                                  | no         |
| `ghostty-dotfiles`          | Mac, macos | `~/.config/ghostty/` settings and themes                                                                                                                                                                                                      | no         |
| `zed-dotfiles`              | Mac, macos | `~/.config/zed/` settings and formatter values                                                                                                                                                                                                | no         |
| `git-signing`               | Mac, macos | GPG commit signing                                                                                                                                                                                                                            | no         |
| `spotlight-exclude`         | Mac, macos | Spotlight exclusions for build folders                                                                                                                                                                                                        | no         |
| `dock-reset`, `dock-spacer` | Mac, macos | the Dock                                                                                                                                                                                                                                      | no         |
| `backup-exclusions`         | Mac        | `~/.cache` and Tart's VMs out of Time Machine (`tmutil addexclusion`) ([Disk](disk.md#backups))                                                                                                                                               | yes        |
| `tart-launcher`             | Mac        | `~/.local/bin/tart`, the launcher that sets `TART_HOME` ([Disk](disk.md#tart))                                                                                                                                                                | no         |
| `tailscale-cli`             | Mac        | `~/.local/bin/tailscale`, a two-line script running the App Store app's CLI; `apps install tailscale` applies it on a fresh install only, else `station settings apply tailscale-cli` ([Machines](machines.md#tailscale)); it needs tailscale | yes        |
| `ssh-key`                   | Mac        | `~/.ssh/id_ed25519`                                                                                                                                                                                                                           | no         |
| `dns`                       | Mac        | the Quad9 encrypted DNS profile                                                                                                                                                                                                               | no         |
| `autologin`                 | Mac        | automatic login, for an unattended Mac; admin                                                                                                                                                                                                 | yes        |
| `restart-after-power-cut`   | Mac        | `pmset` autorestart and wake for network, so the Mac comes back after a power cut; admin                                                                                                                                                      | yes        |
| `never-sleeps`              | Mac        | the engine keeps the Mac awake, its servers with it; no admin, applied from any Mac                                                                                                                                                           | yes        |
| `lock-after-login`          | Mac        | lock the screen after the automatic login; admin                                                                                                                                                                                              | yes        |
| `sshd`                      | Mac        | Remote Login; admin                                                                                                                                                                                                                           | yes        |
| `claude-token`              | Mac        | the Claude Code token shared with VMs ([Accounts](accounts.md#the-vault))                                                                                                                                                                     | yes        |
| `git-shortcuts`             | all        | station's `[alias]` block in `~/.gitconfig`: wip, unwip, sync, feat, fix, chore, branches, lg, st, pushup                                                                                                                                     | yes        |
| `docker-shortcuts`          | all        | `dclean`, `dclean-images`, `dreset` in a block of `~/.zshrc` (`station:docker-shortcuts`), each asks y/n before it removes every container, image, or both                                                                                    | yes        |
| `agent-hooks`               | VMs        | station's hook in `~/.claude/settings.json` (beside the owner's keys) and `notify` in `~/.codex/config.toml` (never one the owner set), for the apps installed of the two; it needs one of them                                               | yes        |
| `herdr-machine`             | Mac        | every workspace of this Mac is a machine its Herdr reaches, by the workspace's name (`hc machine add`); off forgets them, never the owner's other machines; it needs herdr-core, the `hc` that comes with herdr-app                           | yes        |

The settings are Set by station; `autologin`, `restart-after-power-cut`, `never-sleeps`, `lock-after-login` and `sshd` are its Always-on Mac group, for a Mac that hosts servers: opt-in, applied by name only, one folded line until one of it is applied. An admin setting (all of the group but `never-sleeps`) applies only when you run station with admin rights on that Mac's own terminal: without them it is refused, exit 2, with the line that does it, `sudo station settings apply <setting>`, or through `--on`, `ssh -t <mac> sudo station settings apply <setting>`. station never calls sudo itself.

`agent-hooks` is how station knows what an agent does in a workspace: each event of Claude Code's hooks, and each turn Codex completes, runs `station agents record --app <app>` inside the VM, which folds it into the VM's journal of agents (`~/.local/state/station/agents/`) and exits 0 within 200 ms whatever happens, so a hook never blocks or fails the agent. What it keeps of what an agent said is two scrubbed lines, never code, a command or a secret ([ADR-027](../decisions/027-the-agent-hook-keeps-a-scrubbed-line.md)). The guest reports that journal to its Mac's engine at once and on each change; the engine keeps the last report of each VM in its memory and tells once of an agent that starts (`agent_started`, `docs-bot starts on linux-1`), that turns to wait on you (`agent_waiting`, `review-bot is waiting · PR #42 is ready`) or that finishes (`agent_finished`). Without it, station sees only what Herdr says of the agent's terminal: at work, waiting with the program's own words, or running.

## Checked

The Checked group is what station reads from macOS and never applies, on a Mac only (a VM reads each Not for this machine). Each is As expected (`●`) or needs a look (`◐`), its fix the System Settings pane or the command for you to run: station never runs sudo. They are read by `settings/contract.Inspector` (`macos`), each within its budget.

| Setting             | As expected when                                                              |
| ------------------- | ----------------------------------------------------------------------------- |
| `filevault`         | FileVault encrypts the disk                                                   |
| `firewall`          | the macOS firewall refuses unasked connections                                |
| `sip`               | System Integrity Protection keeps the system files read-only                  |
| `automatic-updates` | macOS installs its security responses by itself                               |
| `screen-lock`       | the screen locks after 5 min idle (a desktop: 5 min after the display is off) |
| `gatekeeper`        | Gatekeeper opens only signed apps                                             |
| `remote-login`      | Remote Login is off on a laptop, on on an always-on Mac                       |
| `time-machine`      | Time Machine's last backup is within 7 days                                   |
| `macos-updates`     | no macOS update is waiting                                                    |

## Values

A package with inputs (`autologin`, `claude-token`) or an interactive procedure (`git-signing`, `ssh-key`, `github-login`) runs on a terminal only. A form asks each value, a password is never echoed, and `AGENT_PASSWORD` pre-fills the autologin password. Off a terminal the command exits 2 with `Hint: run it on a terminal`.

There is no flag to pass a value: a secret never travels in argv or shell history ([ADR-009](../decisions/009-one-bootstrap-verb-and-no-values-in-argv.md)).

## Related

- [Commands](commands.md): `station settings`.
- [Preferences](preferences.md): `off_here`, and the environment station reads.
- [Dotfiles](dotfiles.md): the files the dotfile packages copy.
- [Footprint](footprint.md): every file station writes.
