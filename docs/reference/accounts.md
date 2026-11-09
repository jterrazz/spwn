# Accounts

`station accounts` is who is signed in on a machine, and what is shared with its VMs: names and scopes, never a token. A secret travels only as far as its tier allows: kept on the Mac, shared with a VM as a copy for a time, or bound inside one VM.

| Section          | Answers                                                             |
| ---------------- | ------------------------------------------------------------------- |
| The three tiers  | What each tier holds and what delete does to it                     |
| Sharing accounts | The share list, `accounts share` and `take-back`, and when one ends |
| Where to revoke  | Where each provider lets you invalidate a token                     |
| Studio in a VM   | What a share needs inside the VM                                    |
| Claude in a VM   | Logging Claude Code in on a VM                                      |
| Accounts         | Who has access to what, and where each right comes from             |

## The three tiers

| Tier   | Lives                                        | Examples                                                       | What `machines delete` does                                         |
| ------ | -------------------------------------------- | -------------------------------------------------------------- | ------------------------------------------------------------------- |
| kept   | the Mac only                                 | the owner's GitHub token, Apple ID, `AGENT_PASSWORD`           | nothing: it never leaves                                            |
| shared | the Mac; a copy in the VM's credential store | `session:*` browser sessions; the Claude token                 | the copy goes with the disk; the secret stays valid at its provider |
| bound  | inside one VM                                | its ssh host keys, its `github-login` token, a `claude /login` | the disk goes; a token stays valid at its provider until revoked    |

Deleting a VM is not revoking a token. Use fine-grained, expiring tokens in a VM, and revoke at the provider.

## Sharing accounts

`share_list` in `~/.config/station/preferences.json` names the secrets whose accounts may be shared, written by hand: a name, or a family as `session:*`. A Mac whose preferences name none shares `session:*` and `operator:claude:oauth-token`; a v2 `lend_list` is read while no `share_list` is written. An account whose secret the list does not hold is refused (`npm is not on the share list`, exit 2). `station secrets allow` and `disallow` are retired: they exit 2 with the share list as their hint.

```sh
station accounts share github --with api            # for 12h; sharing again resets the term
station accounts share claude --with api --for 4h   # a term from 1m to 7d
station accounts take-back github --from api        # delete the VM's copy, after a y/n (--yes off a terminal)
station accounts                                    # names and scopes, never a value
```

A share is refused before anything is read or copied: an unknown account or VM, a stopped VM (`api is stopped`, hint `station machines start api`), a server (`infra is a server: station shares no account with it`), a VM of another Mac (`ci-1 is on mac-mini: share from mac-mini's accounts`, its hint the same command `--on mac-mini`), an account not signed in on this Mac. An export the studio CLI fails is `GitHub could not be read on This Mac` (exit 1): nothing is copied, no share is kept, and the studio CLI's own words go to `~/.local/state/station/logs/accounts.log`, the file the hint names.

A shared copy is a snapshot: later captures on the Mac do not reach it until you share again. Every share ends by itself: `~/.local/state/station/shares.json` keeps, per VM, each secret shared, when and until when (`accounts --json` lists them under `shares`), and at its until the engine takes the copy back from a VM whose guest is connected (at its start and every minute; a VM still connecting, or stopped past its until, has it taken back as its guest connects, before it is reported ready), with no notification. A take-back that fails keeps its share and is told once an hour. `machines delete` drops the VM's shares with its disk. The shares a v2 VM record held move into `shares.json` once, at `station setup`. Values pass between the credential stores through a pipe, never through argv and never to disk on the Mac. A share and a take-back run inside the VM over the engine's one connection to it, so they need the engine up and the VM's guest connected.

`station secrets`, `secrets lend --to <vm>` and `secrets recall --from <vm>` are hidden aliases for one release: they run `accounts`, `accounts share` (each account the share list holds, or each secret named) and `accounts take-back`, naming the v3 command on stderr; `secrets --json` is `station.accounts.list/v1`.

## Where to revoke

| Secret                               | Revoke at                                                                                      |
| ------------------------------------ | ---------------------------------------------------------------------------------------------- |
| A VM's GitHub token                  | github.com, your account page for developer tokens and authorized apps                         |
| A shared `session:*` browser session | log out of that provider in the Mac's browser; the provider's session list shows the others    |
| The Claude token                     | claude.ai, your account page for sessions and tokens; then `station settings off claude-token` |

## Studio in a VM

A share uses the studio CLI's credential store on both ends (`studio credentials export`, `import`, `delete`), because that store has one owner. A VM without `studio` on its PATH makes `accounts share` fail with a hint to this section. How the studio CLI reaches a VM is not designed yet ([TODO](../../TODO.md)); until then, sharing is judged with stubs.

## Claude in a VM

A VM cannot open a browser on your Mac to log in, so the Mac shares it a token.

1. On the Mac, once: `claude setup-token`. It opens a browser OAuth and prints a long-lived subscription token.
2. `station settings apply claude-token`. A form asks for the token, the OS credential store keeps it as `operator:claude:oauth-token` (in the keychain on a Mac), and the name joins the share list.
3. `station accounts share claude --with <vm>` copies it.
4. Inside the VM, `station shell zsh` exports `CLAUDE_CODE_OAUTH_TOKEN` from the VM's credential store at every shell start. With no `studio`, no name or any failure it prints nothing: a shell start never fails.

`station accounts take-back claude --from <vm>`, the share's end and `machines delete` remove the VM's copy; `station settings off claude-token` deletes the Mac's and takes the name off the share list. Revoke the token itself at claude.ai.

The alternative is a bound login: run `claude /login` inside a permanent workspace. The login lives and dies with that VM and is never shared.

## Accounts

`station accounts` (a machine's Accounts section in the TUI) shows who has access to what: each provider's identity on this Mac, then on each of its VMs, and where the right comes from: `own` (a login done on that machine), `lent` (a shared copy, with its `until`), or both; then `# Shared with VMs`, each share with its VM and its time left. A share in its last hour is `◐ expiring` with the `accounts share` command that renews it; an ended one the engine could not take back is `✗ failing` with the `accounts take-back` command. A VM is read by its own guest over the engine's session, so no provider's output ever leaves the VM: only the parsed identities and scopes do. Each provider's read is an allowlist of named fields (`src/accounts/adapter/cli/parse.go`), and every kept value that looks like a token (`gh*_`, `github_pat_`, `sk-ant-`, `npm_`, `xox*-`, a JWT, 32 and more characters of base64 or hex) is replaced by `?`.

The menu bar's Accounts page (`menubar.accounts`, [API](api.md)) reads every Mac that answers through its own engine. Its facts say how many accounts are signed in somewhere and how many copies are shared. It lists what needs you first: each copy with less than 30 min left, the one ending first first, with Extend (shared again for 4 hours from now), then each sign-in a Mac is out of, This Mac's first, with Sign in, which opens Terminal on the login and reads `Waiting…` while it runs: the menu bar holds its own watch of the page Sign in was pressed on and reads it again every 2 s, for at most 5 min, so the panel closing as Terminal opens, or another page shown, never ends the wait, and the need goes once the read says signed in. Then every copy shared with a VM with its time left over its total, and every account signed in somewhere, once, with the Macs that hold it. git's identity and ssh's keys are no sign-in, so the page leaves them out. Its footer opens the full list in Terminal (`station accounts --all-machines`) and shares one (Lend an account). A Mac's page has the same as its Accounts segment, that Mac's alone: its copies shared with Lend an account under them, then what it is signed in to, under its name, each with the words its provider's read printed, as `src/catalog/accounts.go` orders them: GitHub's login and scopes (`jterrazz · repo, workflow`), Claude's plan (`Max plan`). A part the read did not print is left out, never guessed. A running workspace's page has them as its Accounts segment: the copies its Mac shared with it (`Lent to linux-1`), each with what it lets the workspace do, its time left, Extend and Take back, both run on that Mac, then Lend an account; with none, `No account lent to review-bot`; the footnote names whose accounts they are. Lend an account is a sheet over the page: To the running workspace of an awake Mac (said, not asked, on a workspace's own page; from the fleet page a popup that opens that workspace's sheet), From its Mac, whose accounts that can be lent are picked from a list (one signed out there is disabled, `needs sign-in on mac-mini`), For 4 hr, 12 hr or 7 days, each saying when it ends by day (`Ends at 14:30, then taken back`, `Ends tomorrow at 03:30`, `Ends Saturday 14 Nov`); its button names the account and the time (`Share GitHub for 4 hr`) and runs `accounts.share` on that Mac.

## Related

- [Machines](machines.md): `delete`.
- [Settings](settings.md): `claude-token`.
- [Preferences](preferences.md): `share_list`.
