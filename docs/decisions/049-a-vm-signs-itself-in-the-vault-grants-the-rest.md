# ADR-0xx: A VM signs itself in, and the vault grants the rest

**Status:** Proposed
**Date:** 2026-11-14

## Context

Accounts rest on three tiers: a secret is kept on its Mac, shared with a VM as a copy for a time, or bound inside one VM (`docs/reference/accounts.md`, "The three tiers"). No ADR records them; the page is their only owner. The model has four faults.

- **Two secret stores.** Station's own Keychain service holds the vault's secrets, and the studio CLI's store is the one a share really reads and writes. The studio CLI has no designed way into a VM, so a share has never reached a real VM and is judged with stubs.
- **Two lists for one question.** The share list, `shares.json` and the tiers each say who may hold what, and the `lend` words survive as hidden aliases.
- **A sign-in that cannot be done in a VM.** Sign in opens Terminal on the Mac only; nothing relays a login from a VM.
- **Words that drift**: share, lend, keep, bound, kept, capture.

The detector is the part that works: it reads `gh`, `npm`, `claude` and `wrangler` and parses names and scopes by allowlist. The complexity is in the stores, the lists and the missing relay, not in the detector.

The probe of 2026-11-14 (K0) measured what a better model needs and found most of it present.

## Decision

The owner decided five things on 2026-11-14.

1. **A VM signs itself in by default.** A per-tool detector says whether a machine is signed in; a login script runs the tool's own OAuth or device-code flow inside the VM and relays its link and code to the window. A grant covers only what a VM cannot do itself.
2. **Station's vault is the only owner of the Claude token and of every granted secret.** The dependency on the studio CLI's credential store ends.
3. **There is one vault per Mac, and no sync.** A lost vault is made again, not recovered.
4. **An agent gets only expiring tokens, at most 7 days, and Touch ID is asked for each grant**, whoever asks: the window, the menu bar or the CLI.
5. **The `session:*` browser sessions go**, provided `gh` signs in directly. The probe saw its login print a one-time code and wait for it; a completed login is still open, below.

Three concepts replace the three tiers: a **sign-in** (a tool's login on a machine), a **vault entry** (one secret on the owner's Mac) and a **grant** (an entry given to a machine or an agent until a time). A machine is, per tool, signed in, granted or signed out.

The window owner's review of the same day is binding on the wire's words, and the calls' shapes are the owner of the rest (`apps/engine/remote/api/calls.json`, once the shapes land).

- **`Exchange`, never `Handle`.** A login's steps, `accounts.open` and `accounts.answer` carry an `Exchange`: an opaque id of the login exchange, never a URL. The window holds no URL; the engine of the VM's own Mac opens the link, from an allowlist of hosts.
- **A row never offers what the engine would refuse.** `accounts.list` answers `Action: none` with the engine's own line (`Sign in from mac-mini`) for a VM of another Mac, for a server and for a stopped machine. The refusals stay as guards beneath it.
- **`accounts.logout` exists**, and `Action` gains `sign-out`. The engine words the confirmation: the token stays valid at its provider.
- **`Field` on a refusal.** The declared refusals of `vault.add` and `grants.add` name the input at fault in `Field`: `label`, `value` or `term`.
- **A step carries the engine's sentence** for each kind of step and for an expiry, and `Now` beside `Expires`; on success the sheet shows who signed in.
- **A login for a VM of another Mac is refused in v1.** The Mac that owns the VM does it.
- **One Passwords page shows this Mac's vault**, with no Mac picker.
- **Claude reaches the vault from a sign-in** done on This Mac, whose token goes to the vault. A typed `setup-token` is the fallback only if a capture cannot work.
- **The webview password input is the last fallback** for a typed value, under three conditions: an input that never reaches state, a log or a recording; one frame that is never logged; cleared on send and on close.

What the probe settled:

- **A login streams over the existing `Attach`.** `GuestCommand` already carries stdin and writers and `process.Guest.Attach` already wraps it. Two gaps close: `Attach` discards stderr, where `gh` prints its code, and it returns no exit code, so done and failed are unknown. A line splitter and the call itself are the login lot's work. `engine.guest-exec` stays one-shot.
- **The SIGTERM before the connection closes is load-bearing.** Closing the SSH client without the signal, when no terminal is allocated, left `gh` orphaned and polling. With it, `gh` and `wrangler` (a launcher and a child) both ended. A test pins it.
- **Presence is `LAContext` from the engine**, not an access control on the Keychain item: it carries station's reason, works for a grant the CLI asks, and keeps the item plain. A native password sheet drew from a bare binary run as a LaunchAgent in a user session.
- **Station writes a granted secret into the tool's own store** over stdin, never argv ([ADR-009](009-one-bootstrap-verb-and-no-values-in-argv.md)): `gh auth login --with-token`; a `~/.npmrc` line written with the mode 0600; an env file the shell block sources for the tools that read one. Herdr accepts a per-terminal `env`; station's adapter has no field for it yet, and an agent's grant needs one.
- **The wrangler device page is on `dash.cloudflare.com`**; the other hosts are `github.com`, `www.npmjs.com` and `claude.com`.

Still open, and decided by none of this record:

- **The Touch ID sensor sheet.** The probe VM has no sensor; one real sheet from the real engine LaunchAgent is the only proof.
- **An agent in a Background session.** One that is not in the user's graphical session will, by Apple's rules, get no sheet; untested.
- **The Tauri secure prompt.** No plugin draws a native secure text field; a native alert from Rust or a small helper process are untested routes, and the webview input above is the fallback.
- **The Keychain access-control route**, and **a real login completed** for each tool, which need the owner's browser.

## Consequences

- **Deleted, in the lots that build this:** `accounts.share` and `accounts.take-back` with their nine codes, the settings `claude-token` and `github-login`, `share_list` and `lend_list`, `shares.json`, the studio CLI adapter, `secrets lend` and `recall`, and the share scopes of the catalog. `engine.accounts` stays, reshaped. There is no alias and no reader for what goes (the owner's rule of 2026-11-12: no legacy).
- **Station's Keychain is the one store.** It holds the vault, and the studio CLI's store is read by station no more.
- **Not built.** This record changes no code. `docs/reference/accounts.md` describes the target; until the lots land, the code still has the tiers and the share calls. Other pages that name the share list, `claude-token` or the studio CLI's store (Preferences, Settings, Commands, Machinefiles, API, Architecture) change with the lot that deletes what they describe.
- **A revocation is a cleanup, never a guarantee.** At a grant's end, or at `grants.remove`, station runs the tool's own Remove inside the VM, and an env-file tool leaves new shells only. A token stays valid at its provider; only a short-lived token truly ends.
- **An agent's grant is the riskiest.** An agent, or a prompt injected into it, can send a token it reads anywhere, and [ADR-048](048-the-engine-withholds-a-secret-before-any-answer.md) screens station's answers, not an agent's traffic. Hence an expiring token for an agent (1 minute to 7 days, never until removed), only an entry marked agent-grantable, and Touch ID at each grant.
- **The Machinefile's `accounts.share` and `accounts.keep` become `grants` and `signins`**; which tool's login and injection is the catalog's data (`apps/engine/catalog/accounts.go`), which tools a machine names is the Machinefile's.
- **The vault is lost with its Mac.** The cost of no sync is accepted; every entry is a token that can be minted again.
