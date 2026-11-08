# Ports

`station ports` lists every TCP port that listens on a Mac or a VM, each in the class of who reaches it. A workspace is not on the tailnet, so its Mac shares what it listens on: the Mac's engine reserves a port of its own for it, listens there on 127.0.0.1 only, and carries each connection into the VM over its one ssh connection. The tailnet reaches that port through the Tailscale app alone: `tailscale serve` shares it, https for an HTTP server and raw TCP for any other, so nothing of station listens on the tailnet. Nothing is set per port: a dev server that starts listening in a workspace is shared within seconds. A server's ports are never shared by themselves: its own tooling chooses its routes (its own Tailscale device, a tunnel), and `station ports share infra 8080` shares one on demand.

| Section               | Answers                                                     |
| --------------------- | ----------------------------------------------------------- |
| How a port gets a URL | The guest, the reserved port, the sharing                   |
| The commands          | `ports`, `ports share`, `ports stop-sharing`, `ports https` |
| The policy            | The `ports` group of `preferences.json`                     |
| Serve and HTTPS       | Serve on or off, HTTPS links, the `port` and `path` modes   |
| Profiles              | How Expo and Vite learn their public URL                    |

## How a port gets a URL

Inside each running VM, `station guest` (hidden, run by the Mac's [engine](machines.md#the-engine) over its ssh connection) reports every listening TCP port, loopback included, and the ports Docker maps. The VM never opens a connection to its Mac and never picks a port of the Mac. Ports below 1024 and the ignored ones (22) are never reported.

For each reported port the engine takes a **reserved port**: the lowest free port of the range (default 40000-40999), sticky on (VM, guest port), so the same dev server gets the same URL next week. It then **shares** it: one listener of the Mac on `127.0.0.1:<host port>` (never the tailnet ip, `0.0.0.0` or `::`: station refuses to bind every interface), each connection carried by ssh `direct-tcpip` to the address the VM reported, and, while [Serve](#serve-and-https) is on, a `tailscale serve` entry that exposes that port on the tailnet. A server bound to 127.0.0.1 in the VM needs no `--host`. tailscaled carries the tailnet's bytes, so a Mac's application firewall, which drops an unsigned binary's bytes on the tailnet ip with no prompt, never stands in the way, and who may reach a shared port is the tailnet policy's, never station's.

| Event                                   | What happens                                                        |
| --------------------------------------- | ------------------------------------------------------------------- |
| the port starts listening               | reserved and shared                                                 |
| it listens in a server                  | listed, never shared by itself                                      |
| it stops listening                      | unshared after 5 s (a restart inside them keeps it); reserved kept  |
| a VM stops, then runs again             | unshared, then shared again on the same port of its Mac             |
| `machines delete`, a throwaway poweroff | released, its serve entry removed                                   |
| `station ports stop-sharing`            | released and remembered: not shared again until a share             |
| 7 days without listening                | released, unless pinned by a [profile](#profiles)                   |
| the range is full, or the VM holds 20   | waiting, `◐` with the cause and the fix                             |
| the port of the Mac is busy             | the next free port (a pinned one waits, retried every 10 s)         |
| the Mac reboots                         | the engine shares every reserved port again as the guests report    |
| a server goes down                      | its ports stay listed, `unreachable while infra is down`, never red |
| a server boots again                    | its ports listed `starting` until its guest reports them            |

The reserved ports live in `~/.local/state/station/leases.json`, a kept file: deleting it changes every URL. Each has two URLs, one shape each: `local` `http://127.0.0.1:<port>` (no scheme for a raw TCP server) and `public`, the one `ports share` prints and a profile exports. With Serve on, `public` is `https://<public host>:<port>` for an HTTP server or one not yet detected (per [mode](#serve-and-https)) and `<public host>:<port>` for a raw TCP one; with Serve off it is the local URL, the only one there is. `<public host>` is the Mac's MagicDNS name.

## The commands

```sh
station ports                        # the Tailscale line, then every listening port in its class
station ports --all-machines         # every Mac's, asked over ssh
station ports --all                  # the ports that listen on their machine only, too
station ports --json                 # station.ports.list/v1
station ports share api 3000         # reserve 3000's URL now and print it; in a VM: station ports share 3000
station ports stop-sharing api 3000  # on a Mac; in a VM: station ports stop-sharing 3000
```

The list begins with the Tailscale line, then every port that listens on a Mac or a VM, in the class of who reaches it, piped tab-separated, each row saying its class in words, so a `grep` of one port still tells Private from Public:

| Section                     | Holds                                                                                                              | Columns                                                           |
| --------------------------- | ------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------- |
| Needs you (`✗`)             | a port reachable outside Tailscale (every interface, the LAN) that no declaration names public                     | machine, port, `needs you`, process, bind, `declared no`, the fix |
| Public (`●` `◐`)            | a port a declaration names public, `●` when its route is seen, `◐` with what is missing when it is not             | machine, port, `public`, process, bind, the route and its proof   |
| Private (`●` `◐` `○`)       | what is shared with you on Tailscale: a VM's reserved port, a machine's own Tailscale device                       | machine, port, `private`, URL, state, process, profile, fix       |
| Listens on its machine only | loopback and other ports only their machine reaches; one `N more listen on their machine only` line unless `--all` | machine, port, `this machine only`, process, bind                 |

A VM behind its Mac is reached through Tailscale only, so its ports are Private whatever it binds. Among the shared rows `●` is shared, `○` not listening (reserved, nothing listens), `◐` waiting (the VM shares 20 already, the range is taken), local only (Serve off, Tailscale signed out, a refused serve), `issuing certificate`, not shared while its VM is stopped or while the engine is stopped. A Mac that does not answer is one `◐` row, never a failure. In `--json` a shared port carries `shared_at`, when the owner shared it (UTC): the menu bar marks a share of the last 8 s fresh.

`station ports declare <machine> <port> --via <route> [--url <link>]` declares a port public on the Mac that keeps the machine's declarations (`~/.config/station/public.json`, a VM's on its Mac), wherever it is asked from; the route is how the port is reached from elsewhere, and what the list looks for to call it seen:

| Route               | Seen when                                                                                                             |
| ------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `cloudflare-tunnel` | a `cloudflared` connector runs on that machine (its metrics listener; inside a VM, as its guest reads it)             |
| `tailscale-funnel`  | the Mac's Funnel state (`tailscale serve status --json`, `AllowFunnel`) lists the port; a VM's own device is not read |
| `lan`               | the port listens where the machine's network reaches it: every interface or a LAN address                             |

No `--via` and an unknown route are refused (exit 2) with the three routes. `undeclare` asks yes or no, then the port is red again while it is still reachable outside Tailscale; a port never declared is refused (exit 2). Inside a VM both are refused: its Mac keeps them. The Remote login setting (`sshd`), applied on a Mac with no battery (an always-on Mac, never a laptop), declares its `22` through `ports.declare` (`your LAN · Remote login · by Settings`), and `settings off sshd` takes it back. `--url` keeps the link the port is reached by from outside (`https://jterrazz.com`): the menu bar's public row names its domain beside the route, opens it, copies it and shows it as a QR; without one the row copies the port's address. In `--json` a declared port carries `declared: {via, url, by, seen, missing}`.

Each Mac's reserved ports are one line at the end: `Reserved on This Mac: 2 of 1000 (40000-40999), 998 free`, one per Mac with `--all-machines`. In `--json` each Mac carries it in `range` (`mac`, `from`, `to`, `reserved`, `free`), read from its policy and reservations file, so it is there with the engine down.

The engine's exposure job reads what listens on its Mac and its VMs at its start, every 5 minutes and whenever a guest reports a change. A port that turns red is a finding of `station overview` (`✗ port 9000 is reachable outside Tailscale`) and is told once, as a `port_exposed` event (`<machine>: port <port> is reachable outside Tailscale`), then again only after it stopped being red.

`ports share <port>` in a VM reserves the port at once, whether or not anything listens yet, so a value can be computed before the server starts: `$(station ports share 3000)`. On a Mac it names its VM, `station ports share api 3000`; with the Mac's engine away it is exit 1 (`Error: macbook is not connected to this VM`, `Hint: on macbook: station setup`), and in a VM no Mac has connected to yet, so with no Mac name to give, every `ports` command is exit 1 (`Error: no Mac has connected to this VM yet`, `Hint: on the VM's Mac: station setup`).

`ports stop-sharing` gives a reserved port back, after yes or no (`--yes` piped), and is remembered, so the next listen is not shared until `ports share`. A VM of another Mac is released on that Mac, over ssh. With the engine down, a Mac edits the reservations store itself.

## The policy

The Mac owns it, in `preferences.json`; every key is optional and only `https` has a command, [`ports https`](#serve-and-https):

```json
"ports": { "range": "40000-40999", "max_per_vm": 20, "ignored_ports": [22], "min_port": 1024, "https": "port" }
```

The engine reads the file again when it changes. [Preferences](preferences.md) has the rest of `preferences.json`.

## Serve and HTTPS

Every shared port leaves the Mac through Tailscale Serve. Serve is **on** when `tailscale status --json` lists a certificate domain (`CertDomains`: the admin has enabled HTTPS certificates, which Serve needs); the tailnet logged out or absent is off. The engine reads it at its start, at each VM run and every 30 s; `station status` shows it as the `Tailscale Serve` gateway and `ports --json` as each Mac's `serve` (`on` or `off`).

To turn it on, once per tailnet: in the [admin console's DNS page](https://login.tailscale.com/admin/dns), MagicDNS on, then HTTPS Certificates (machine names then appear in Certificate Transparency logs), and Serve where the console asks for it.

With Serve on, an HTTP server (detected once per reserved port: a server that speaks first is not HTTP, else `HEAD /` must answer `HTTP/`) is served https with a valid certificate, so an iPhone on the tailnet opens it as a secure context; any other TCP server is served raw:

| Port                 | Serve line                                                                              | Public URL                                  |
| -------------------- | --------------------------------------------------------------------------------------- | ------------------------------------------- |
| HTTP, `https` `port` | `tailscale serve --bg --https=<host port> http://127.0.0.1:<host port>`                 | `https://macbook.tail1234.ts.net:40000`     |
| HTTP, `https` `path` | `tailscale serve --bg --https=443 --set-path=/<vm>/<port> http://127.0.0.1:<host port>` | `https://macbook.tail1234.ts.net/api/8081/` |
| raw TCP              | `tailscale serve --bg --tcp=<host port> tcp://127.0.0.1:<host port>`                    | `macbook.tail1234.ts.net:40002`             |

**HTTPS links** are the Mac's choice, `station ports https on|off [--on <mac>]` (the call `ports.set-https`, refused inside a VM: its Mac serves its shares), kept as `https` in its `ports` group. Off (`"https": "off"`), every share is served raw, `tailscale serve --bg --tcp=<host port> tcp://127.0.0.1:<host port>`, an HTTP server under `http://macbook.tail1234.ts.net:40000` and no certificate asked for; on keeps a `path` mode, else `port`. The engine serves its shares again in the new mode at its next tick, each unshared first in the mode it was served in, and at its start removes an entry served in the other one.

Unsharing runs the same line ending `off`. Any other word of `https` is read as `port`. A serve refused while Serve is on leaves the port shared locally: its row `◐` with `serve:` and tailscale's own line (never the CLI's `Warning:` lines), its public URL unchanged, the serve retried at its next share. In `path` mode a server that builds absolute URLs needs its base path set to `/<vm>/<port>/`.

tailscaled issues the Mac's certificate at the first https request, 34.5 s measured, so the engine warms it: at its start with Serve on, and when Serve turns on, it runs `tailscale cert --cert-file=- --key-file=- <host>` in the background, its output thrown away (`/dev/null` is refused as not a regular file), one at a time, within 120 s, never holding a share. Meanwhile every row served `--https` says `issuing certificate` (`◐`, its URL unchanged); `engine.log` then says `certificate for <host> ready in <d>` or `not issued:` with tailscale's line.

With Serve off no serve runs (one would block waiting for the admin): every reserved port is local only, every row of the Mac says `Serve is off on Tailscale: turn on HTTPS certificates in the Tailscale admin console`, and every public URL is the local one; `engine.log` says it once per change. Serve turning on serves every shared port at the next read.

Station owns only the serve entries on its range (both kinds) or under `/<vm>/` of its VMs (`path`), and, while Serve is on, removes the ones no reserved port justifies at the engine's start and when Serve turns on; it never touches another entry.

## Profiles

A server that must know its own public URL gets it from its environment. A profile, data in the catalogue, names a dev server's default port and the variables that carry the URL; at `machines create` the engine reserves every profile's port for the new VM, and the VM's shell (`station shell zsh`, from its ports' env file) exports the variables. The engine renders them at every run of the VM (create, `start`, `restart`, its own run at boot; never when it adopts a running VM) from Serve's state then, so they are fixed for a run, and a permanent VM picks up the public URLs at its first start after Serve is turned on:

| Profile  | Guest port | Variable                                 | Value                                                   |
| -------- | ---------- | ---------------------------------------- | ------------------------------------------------------- |
| expo     | 8081       | `EXPO_PACKAGER_PROXY_URL`                | the public URL, `https://macbook.tail1234.ts.net:40000` |
| vite     | 5173       | `__VITE_ADDITIONAL_SERVER_ALLOWED_HOSTS` | the public name, `macbook.tail1234.ts.net`              |
| every VM |            | `STATION_PUBLIC_HOST`                    | the public name                                         |

With Serve off they hold the local URL, `http://127.0.0.1:<host port>`, and the public name `127.0.0.1`, and `machines create` says so on a `◐ ports` row. So `npx expo start` in a VM writes the shared https URL into its manifests, and Vite accepts requests for the public name, with no flag. A server on another port composes its URL itself: `PORT=3000 BASE_URL=$(station ports share 3000) npm start`. Same-origin traffic (HMR websockets, relative URLs) needs nothing: it rides the same port of the Mac. A server whose setting has no environment variable gets its line here.

## Related

- [Machines](machines.md): the engine, and how a VM is reached by ssh.
- [The API](api.md): the engine's calls `ports.share` and the reservations ride.
- [Preferences](preferences.md): `preferences.json`.
- [Glossary](../concepts/glossary.md): port, reserved port, share, profile, public name.
