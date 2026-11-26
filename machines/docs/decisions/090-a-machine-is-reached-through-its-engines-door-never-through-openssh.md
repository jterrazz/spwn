# ADR-090: A machine is reached through its engine's door, never through OpenSSH

**Status:** Accepted
**Date:** 2026-11-24

Amends [ADR-016](016-api-v2-and-remote-by-api.md), [ADR-064](064-a-key-never-leaves-its-device.md) and [ADR-074](074-a-far-station-answers-its-hello-within-5-seconds.md).

Amended by [ADR-092](092-a-machines-device-book-says-who-may-reach-it.md): the gesture is `spwn devices add`, not `allow`, and `hello` is no channel.

Accepted by the owner on 2026-11-24, on the owner's word, on one condition: nothing of it is built before the probes PT1 to PT5 and PH1 to PH3 pass.

Amended by [ADR-096](096-spwn-is-one-app-its-menu-bar-notifications-and-engine-live-while-it-runs.md) (Proposed): the client's presence is asked once per Spwn launch, and the door is up while Spwn is open.

## Context

Another Mac is reached by `ssh -o BatchMode=yes -- <mac> station engine attach`, bridged to its engine's socket ([ADR-016](016-api-v2-and-remote-by-api.md)). It needs Remote Login and an `authorized_keys` line on every far Mac. A VM is reached through its Mac's engine, in process, by `golang.org/x/crypto/ssh` to the VM's sshd, and is never on Tailscale. The owner's own `ssh`, editors and Herdr ride managed `~/.ssh/config` blocks whose ProxyCommand is `station machines pipe <vm>`. The engine listens on its unix socket only, held by the rule `engine-no-tcp`.

The owner asked for the opposite, in the audit (2026-11-24): « spwn devrait se connecter aux autres ordis par le binaire qui expose le service spwn nn ? et rien de ssh à supporter etc etc. »

[ADR-064](064-a-key-never-leaves-its-device.md) weighed "a door of station's own (an mTLS port on the tailnet)" and refused it: it trades OpenSSH's audited auth for code station would write, and a shell would then depend on the engine. The options weighed for this record:

- Keeping OpenSSH and Remote Login is the very thing to remove.
- An mTLS or Noise door is crypto and auth written here, with no client for Herdr's phone app or the editors: 064's objection holds in full.
- An embedded `tsnet` node is what [ADR-010](010-vms-are-reached-through-their-mac.md) retired: minted auth keys and a second node per Mac.
- Tailscale's identity alone, on a plain listener, makes any node of the tailnet the owner, with no presence and no use of the device keys.
- Riding Herdr's machine link reverses the dependency of [ADR-089](089-an-agent-runs-in-a-herdr-terminal-on-its-own-machine.md), and that link is itself OpenSSH.

## Decision

- **Each engine opens one door**: an SSH-protocol server inside the engine, `golang.org/x/crypto/ssh` (already a dependency), bound to the machine's Tailscale address only, read at runtime, never a wildcard, on one fixed port.
- **The door serves no shell.** It opens four named channels and refuses everything else, `shell`, `pty-req`, an arbitrary `exec` and port forwards included:

    | Channel     | Carries                                                                    |
    | ----------- | -------------------------------------------------------------------------- |
    | `spwn-api`  | the engine's API frames into its local socket, in place of `engine attach` |
    | `pipe <vm>` | bytes to the sshd of a VM behind this Mac, for editors and `git`           |
    | `herdr`     | bytes to this machine's `herdr-core connect`                               |
    | `hello`     | liveness, inside ADR-074's 5 s budget                                      |

- **064's objection is answered, not ignored.** The protocol and its auth are `x/crypto/ssh`, not code written here; the one policy written here is "this public key is in the device book". And no shell depends on the engine, since the door serves none.
- **The client signs with its device key** through the engine's own agent (ADR-064, as amended by [ADR-076](076-spwn-on-a-machine-one-active-app-one-background-job-the-engine-updates.md)), presence asked once per engine life; a Mac run alone signs with [ADR-085](085-a-mac-run-alone-keeps-a-device-key-that-signs-with-nobody-present.md)'s key.
- **The door accepts only the keys of its device book**, the book [ADR-077](077-a-workspace-trusts-its-macs-device-key-first.md)'s lot L3 builds, which is built first. **A device is added only with the owner's Touch ID on the machine that accepts it** (`spwn devices allow`).
- **The door's host key is an Enclave key of the engine**, pinned on the client under `~/.local/state/station/known_hosts/<machine>`.
- **A machine is found among the tailnet's peers** that `tailscale status --json` gives, and is a `spwn` machine once its door answers `hello`. Tailscale still crosses NAT; VMs stay off the tailnet and are reached through their Mac's door, then its guest connection ([ADR-016](016-api-v2-and-remote-by-api.md) holds).
- **A far agent's terminal reaches this screen through Herdr's own relay**, over the door's `herdr` channel: this Mac's Herdr holds the machine with `--via 'spwn machines pipe --herdr <machine>'`, kept by a setting of the herdr-core package.
- **Remote Login is turned off on every Mac.** The iPhone's Herdr app goes through the door if the app allows it (PT3); otherwise Remote Login stays for it alone, as a package.
- **Editors and `git` keep a managed `~/.ssh/config` block** whose ProxyCommand is `spwn machines pipe <vm>`, riding the door's `pipe` channel: ssh on the client, nothing to configure.
- **The rule `engine-no-tcp` becomes `engine-door-only`**: exactly one listener, in `remote/adapter/door`, bound to a Tailscale address.
- **The cut is whole.** `remote/adapter/sshstdio`, the `engine attach` leaf and their specs are deleted, and so are the Mac `Host` blocks and their one-hop ProxyCommand. No compatibility path is kept.

### The owner's words

On Herdr and the door (2026-11-24):

| Question                                                                                                                                                      | Answer                |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------- |
| H3. Each engine opens an SSH door of its own (`x/crypto`, not OpenSSH), on Tailscale only, with no shell; Remote Login is cut on every Mac. After the probes? | « Oui, après sondes » |
| H4. A new device is accepted only with your Touch ID on the machine that accepts it?                                                                          | « Oui »               |
| H5. The iPhone's Herdr goes through the door if its app allows, otherwise Remote Login stays, as a package?                                                   | « Oui »               |
| H6. Editors keep a `~/.ssh/config` block through `spwn machines pipe`?                                                                                        | « Le garder »         |

## Consequences

- ADR-016's "between Macs there is ssh only, and the engine listens on no TCP port" no longer holds; the rest of 016 does.
- ADR-064's refusal of a door of its own is answered by this record, and its key agent's relay of the owner's key leaves with the device book.
- ADR-074's ssh keepalive (`ServerAliveInterval=2`, `ServerAliveCountMax=2`) becomes the door's, in `x/crypto/ssh`; its 5 s hello budget is unchanged.
- The development VM's dedicated file key for the studio's unattended tooling is untouched: it opens that VM's sshd, never a door.
- The rename lot follows the door's, so `spwn machines pipe` is born with its final name.
- Probes owed before any build, each an evidence note:
    - PT1: a listener on the Tailscale address of Spwn.app's engine on macOS asks no Application Firewall or Local Network question on a headless Mac, or once per bundle.
    - PT2: the Enclave P-256 key authenticates to the door through the engine's agent, presence once per engine life, and ADR-085's key signs with the screen locked.
    - PT3: the Herdr iPhone app takes a custom port, offers a key the door accepts, and runs a line the `herdr` channel can serve.
    - PT4: Zed's remote and `git` reach a far VM through `ProxyCommand spwn machines pipe <vm>` over the door.
    - PT5: when the Tailscale address comes late, changes, or Tailscale signs out, the door rebinds and `--on` says the machine does not answer.
    - PH1 and PH2, ADR-089's; and PH3, Herdr's `--via` bridge: add, reconnect after a cut, attach, and a clear error from `machine update`.
