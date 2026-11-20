# Accounts

`station accounts` is who is signed in on a machine, and what the vault granted it: names and scopes, never a token. A VM signs itself in by default; a grant, a secret of the Mac's vault given until you remove it, covers only what a VM cannot do itself. The decision is [the accounts ADR](../decisions/049-a-vm-signs-itself-in-the-vault-grants-the-rest.md).

| Section          | Answers                                                              |
| ---------------- | -------------------------------------------------------------------- |
| What is built    | What this page describes that the code does not do yet               |
| Three concepts   | Sign-in, vault entry and grant, and what a machine shows per program |
| Signing in       | A login run inside the VM, and how its link and code reach you       |
| The vault        | One per Mac: what it holds, how a secret enters, Touch ID            |
| Grants           | What a grant gives, how it arrives, how it ends                      |
| Where to revoke  | Where each provider lets you invalidate a token                      |
| Reading accounts | How a machine's sign-ins are read, and what never leaves it          |
| Windows          | Where each of these shows                                            |

## What is built

This page is the target. Reading who is signed in (`accounts.list`), the redaction of what is read, the list's pages, the login relay, `accounts.logout`, the vault calls, grants to a machine, and the presence check are built. A sign-in's token is taken into the vault for `gh` alone. The Machinefile's `signins` and `grants` are built too: a workspace's first start grants each program of `grants` from this Mac's one password for it and leaves each program of `signins` it is not signed in to with its `station accounts login` command ([the Machinefile's keys](machinefiles.md#the-keys)). A share made before this is never taken back: none had reached a real VM ([ADR-049](../decisions/049-a-vm-signs-itself-in-the-vault-grants-the-rest.md)).The Passwords page is not built yet.

## Three concepts

- **Sign-in**: a program's login on one machine, done by the program itself. The catalog owns how each program is read, logged in and logged out (`apps/engine/catalog/accounts.go`); a Machinefile names only which programs a machine signs in to.
- **Vault entry**: one secret on the owner's Mac, with a label, the program it is for, what kind it is and who it signs in as.
- **Grant**: an entry given to a machine, until it is removed.

Per program, a machine is **signed in** (`How: own`), **granted** (`How: granted`, with `Entry` and `EntryLabel` naming the password) or **signed out**. A machine's own sign-in is never a grant: a grant to a VM already signed in is refused.

A row offers only an action the engine would accept: sign in, sign in again, remove, sign out, or none. For a VM of another Mac, a server or a stopped machine the action is none, and the row carries the engine's line (`Sign in from mac-mini`).

## Signing in

A sign-in is a login the program runs inside the machine. `accounts.login` is a call of the engine of the machine's own Mac, naming the VM in `Machine`: that engine starts the program over its one connection to the VM, reads its output and its errors as they arrive, and streams a step to the window. It runs the program found on the machine's `PATH`, never a version manager's shim, so cancelling the login stops the program itself (a SIGTERM, then the connection's close):

- a **device code**: the window shows the code, and Open takes you to the provider's device page; its status line reads `Enter this code at github.com. linux-1 signs in by itself once you approve it.`;
- a **link**: the VM's own link, the first the program prints on its site or a subdomain of it (another link the program prints is skipped), opened by the engine of the VM's Mac, only to a site the catalog allows; a program that prints links only off its site is refused;
- a **paste**: the program asks for a code you copy back, which `accounts.answer` writes to the program's input, one line, never to a command line. A program that would open a browser finds `true` as its browser, which opens nothing and leaves nothing behind.

The window holds no URL. A step carries an `Exchange`, an opaque id of the login, and `accounts.open` and `accounts.answer` take it back. Link and code are the screened words of [ADR-048](../decisions/048-the-engine-withholds-a-secret-before-any-answer.md); the engine opens a link itself rather than send it. A step is titled "Sign in to <program> on <machine>", the program by its wire name and the machine by its name (This Mac's own for a login of this Mac). A login that ends says who signed in. A VM of another Mac is not signed in from this one in v1: do it from its Mac.

Sign out runs the program's own logout (`accounts.logout`) after a confirmation the engine words: the token stays valid at its provider.

## The vault

The vault is the Keychain entries of one Mac under Machines' own Keychain name, each value under `vault:<id>`, and a book beside them, `~/.local/state/station/vault.json`, that lists each entry and its grants without a value. It has no sync: each Mac has its own, and the Passwords page shows This Mac's (`Passwords on <mac>`), with no Mac picker. Machines is the only owner of the Claude token and of every granted secret; the studio CLI's credential store is not read.

The verdict `vault.list` answers counts the entries (`3 kept, 1 granted`) and says under it, in its `Line`, that a value never leaves this Mac's Keychain to be shown.

Each entry `vault.list` answers says what its removal ends, so the confirmation can word it: `RemoveSentence`, counted from its grants (`Its grants end: 2 machines`, or `Nothing else ends` with none), and `Reach`, `granted` or `not granted`. A grant is only its target; it has no state, since it stays until it is removed.

The Add a password sheet reads one watched call, `vault.new`, of This Mac's engine (a server or a workspace is refused as not for this machine). It answers the `Intro` ("It is kept in <mac>'s Keychain; you can then grant it to a machine."), the `Captures` and the typed way's `Programs`; the sheet words nothing itself. A password enters in one of two ways:

- **From a sign-in** done on This Mac, whose token goes to the vault: the token the program's own command prints (`gh auth token`), read from its output and never shown. Only `gh` has such a command, so `Captures` holds one row per program this Mac is signed in to that has one. A row names the program, who it is, its scopes, the kind the token is kept as (`oauth-token` for `gh`), a sentence that says so ("Kept as an OAuth token, the kind gh holds") and the label proposed for it ("GitHub · you"). With no row, `CaptureLine` says to sign in first, and the way stays on the sheet. The kind is the engine's: `vault.add` from a sign-in derives it and ignores the one sent, and any other program is refused `vault.cannot-capture`.
- **Typed**, as a value, of a kind the program takes. `Programs` lists every program in the catalog's order with the `Kinds` a typed password of it may be and the one offered first: `gh` a fine-grained token or a token, `npm` and `wrangler` a token or a fine-grained token, `claude` an OAuth token. A typed password sends its kind; a request schema never requires a key, so `vault.add` refuses a typed one without it as `call.bad-input`. The value travels as a write-only value, never in an argument list ([ADR-009](../decisions/009-one-bootstrap-verb-and-no-values-in-argv.md)); a native secure prompt asks for it where one can be drawn, and the window's password input is the last fallback, never reaching state, a log or a recording. Whether a native prompt can be drawn is open ([ADR](../decisions/049-a-vm-signs-itself-in-the-vault-grants-the-rest.md)).

A blank label is refused `vault.empty-label`, and a typed password with no value `vault.empty-value`; both carry `Field`.

Each grant asks for Touch ID first, the engine's own `LAContext` with Machines' reason, from the window, the menu bar or the CLI, after the checks and before the Keychain is read. Removing an entry ends the grants made from it, after a confirmation that names them.

## Grants

A grant gives one entry to one machine. It has no end of its own: the owner decided on 2026-11-14 that no timing logic expires an account, so a grant stays until `grants.remove`, the removal of its password or the deletion of its VM. Granting again replaces it.

- **It arrives** over stdin, never an argument: a login with the token for `gh`, a `~/.npmrc` line (mode 0600) for npm, an env file under `~/.config/station/grants/` that the shell block sources for Claude and wrangler.
- **It leaves** at `grants.remove`, or when the VM's guest connects after a removal it could not receive, by the program's own Remove: a logout, or the deletion of the line or file that holds the secret. A program that reads an env file stops in new shells only, and the sentence of `grants.remove` says so.
- **It is refused**, before anything is read, for a server, a VM of another Mac, a stopped VM, a VM already signed in, and a secret that cannot be delivered. The refusals of `vault.add` carry `Field` (`label` or `value`) so a form marks the input at fault; those of `grants.add` name none.

Deleting a VM ends its grants with its disk. It is not a revocation: see below.

## Where to revoke

A grant's removal undoes it inside the VM; a token stays valid at its provider until revoked there. A grant never ends on its own, so keep fine-grained tokens and revoke at the provider.

| Secret              | Revoke at                                                              |
| ------------------- | ---------------------------------------------------------------------- |
| A VM's GitHub token | github.com, your account page for developer tokens and authorized apps |
| The Claude token    | claude.ai, your account page for sessions and tokens                   |

## Reading accounts

`station accounts` is `accounts.list` of one machine: each program's state, who, its scopes, and how, `own` or `granted` with the password's label. A VM's programs are read by the engine of its Mac, which runs each program's read over the VM's one connection and knows what it granted. Each read is the catalog's (`apps/engine/catalog/accounts.go`, a program's `Reads`): a command that never prints a secret, and only the fields it names are kept (`apps/engine/accounts/adapter/cli/signins.go`); any kept value shaped like a secret is replaced by `?`, the shapes being `apps/engine/core/secret`'s, owned by ADR-048.

## Windows

A sign-in that no longer works is a Needs-you finding whose fix is the call `accounts.login` with the program as `Program`, run at that machine; it prints as `station accounts login <program> --on <machine>`. The window's Accounts view has a Sign-ins tab per machine, the login sheet, the Passwords page and the grant picker. The menu bar's Accounts page, which lists what needs you first, offers Grant. Their screens are the window's and the menu bar's own; this page owns the words they show.

## Related

- [Machines](machines.md): `delete`.
- [Machinefiles](machinefiles.md): the `signins` and `grants` keys.
