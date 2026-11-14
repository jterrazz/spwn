# Accounts

`station accounts` is who is signed in on a machine, and what the vault granted it: names and scopes, never a token. A VM signs itself in by default; a grant, a secret of the Mac's vault given for a time, covers only what a VM cannot do itself. The decision is [the accounts ADR](../decisions/049-a-vm-signs-itself-in-the-vault-grants-the-rest.md).

| Section          | Answers                                                              |
| ---------------- | -------------------------------------------------------------------- |
| What is built    | What this page describes that the code does not do yet               |
| Three concepts   | Sign-in, vault entry and grant, and what a machine shows per program |
| Signing in       | A login run inside the VM, and how its link and code reach you       |
| The vault        | One per Mac: what it holds, how a secret enters, Touch ID            |
| Grants           | What a grant gives, how it arrives, how it ends                      |
| Agents           | Why an agent's grant is the narrowest                                |
| Where to revoke  | Where each provider lets you invalidate a token                      |
| Reading accounts | How a machine's sign-ins are read, and what never leaves it          |
| Windows          | Where each of these shows                                            |

## What is built

This page is the target. Reading who is signed in, the redaction of what is read, and the list's pages are built. Everything else here is not: the login relay, the vault calls, grants, `accounts.logout`, Touch ID, the Passwords page and the Machinefile's `signins` and `grants`. Until those land, the code still has the tiers and `accounts share`, which the ADR deletes; [its consequences](../decisions/049-a-vm-signs-itself-in-the-vault-grants-the-rest.md#consequences) list what goes, and a page that still names `share_list` or `claude-token` describes that old code.

## Three concepts

- **Sign-in**: a program's login on one machine, done by the program itself. The catalog owns how each program is read, logged in and logged out (`apps/engine/catalog/accounts.go`); a Machinefile names only which programs a machine signs in to.
- **Vault entry**: one secret on the owner's Mac, with a label, the program it is for, what kind it is and who it signs in as.
- **Grant**: an entry given to a machine or an agent until a time.

Per program, a machine is **signed in** (`How: here`), **granted** (`How: granted`, with its `Until`) or **signed out**. A machine's own sign-in is never a grant: a grant to a VM already signed in is refused.

A row offers only an action the engine would accept: sign in, sign in again, remove, sign out, or none. For a VM of another Mac, a server or a stopped machine the action is none, and the row carries the engine's line (`Sign in from mac-mini`).

## Signing in

A sign-in is a login the program runs inside the machine. The engine starts it over its one connection to the VM, reads the program's output as it arrives, and streams a step to the window:

- a **device code**: the window shows the code, and Open takes you to the provider's device page;
- a **link**: the VM's own link, opened by the engine of the VM's Mac, only to a site the catalog allows;
- a **paste**: the program asks for a code you copy back, which `accounts.answer` carries.

The window holds no URL. A step carries an `Exchange`, an opaque id of the login, and `accounts.open` and `accounts.answer` take it back. Link and code are the screened words of [ADR-048](../decisions/048-the-engine-withholds-a-secret-before-any-answer.md); the engine opens a link itself rather than send it. A login that ends says who signed in. A VM of another Mac is not signed in from this one in v1: do it from its Mac.

Sign out runs the program's own logout (`accounts.logout`) after a confirmation the engine words: the token stays valid at its provider.

## The vault

The vault is the Keychain entries of one Mac under station's own Keychain name. It has no sync: each Mac has its own, and the Passwords page shows This Mac's (`Passwords on <mac>`), with no Mac picker. Station is the only owner of the Claude token and of every granted secret; the studio CLI's credential store is not read.

A secret enters in one of two ways:

- **From a sign-in** done on This Mac, whose token goes to the vault. Claude's does.
- **Typed**, as a value. It travels as a write-only value, never in an argument list ([ADR-009](../decisions/009-one-bootstrap-verb-and-no-values-in-argv.md)); a native secure prompt asks for it where one can be drawn, and the window's password input is the last fallback, never reaching state, a log or a recording. Whether a native prompt can be drawn is open ([ADR](../decisions/049-a-vm-signs-itself-in-the-vault-grants-the-rest.md)).

Each grant asks for Touch ID first, the engine's own `LAContext` with station's reason, from the window, the menu bar or the CLI, after the checks and before the Keychain is read. Removing an entry ends the grants made from it, after a confirmation that names them.

## Grants

A grant gives one entry to one machine or one agent, for a term. Granting again starts the term over.

- **It arrives** over stdin, never an argument: a login with the token for `gh`, a `~/.npmrc` line (mode 0600) for npm, an env file under `~/.config/station/grants/` that the shell block sources for Claude and wrangler.
- **It leaves** at `grants.remove`, at its `Until`, or when the VM's guest connects after it, by the program's own Remove: a logout, or the deletion of the line or file that holds the secret. A program that reads an env file stops in new shells only, and the sentence of `grants.remove` says so.
- **It is refused**, before anything is read, for a server, a VM of another Mac, a stopped VM, a VM already signed in, a term out of bounds, and a secret that cannot be delivered. The refusals of `vault.add` and `grants.add` carry `Field` (`label`, `value` or `term`) so a form marks the input at fault.

Deleting a VM ends its grants with its disk. It is not a revocation: see below.

## Agents

An agent's grant reaches the environment of its own terminal, not the VM's shell, and ends when that terminal stops or at its `Until`. It is the narrowest:

- only an entry marked agent-grantable, a fine-grained, expiring token, never the owner's own GitHub login;
- a term from 1 minute to 7 days, never until removed;
- Touch ID at each grant.

An agent, or a prompt injected into it, can send a token it reads anywhere; station screens its own answers, not an agent's traffic.

## Where to revoke

A grant's end undoes it inside the VM; a token stays valid at its provider until revoked there. Use fine-grained, expiring tokens, and revoke at the provider.

| Secret              | Revoke at                                                              |
| ------------------- | ---------------------------------------------------------------------- |
| A VM's GitHub token | github.com, your account page for developer tokens and authorized apps |
| The Claude token    | claude.ai, your account page for sessions and tokens                   |

## Reading accounts

`station accounts` shows who has access to what: each provider's identity on this Mac, then on each of its VMs, and where it comes from, `here` (a sign-in done on that machine), `granted` (with its `until`), or both. A grant in its last hour is `◐ expiring`; one whose end the engine could not carry out is `✗ failing`. A VM is read by its own guest over the engine's session, so no provider's output leaves the VM: only the parsed identities and scopes do. Each read is an allowlist of named fields (`apps/engine/accounts/adapter/cli/parse.go`), and any kept value shaped like a secret is replaced by `?`; the shapes are `apps/engine/core/secret`'s, owned by ADR-048.

## Windows

The window's Accounts view has a Sign-ins tab per machine, the login sheet, the Passwords page and the grant picker. The menu bar's Accounts page, which lists what needs you first, offers Grant. Their screens are the window's and the menu bar's own; this page owns the words they show.

## Related

- [Machines](machines.md): `delete`.
- [Machinefiles](machinefiles.md): the `signins` and `grants` keys.
